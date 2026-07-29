import assert from 'node:assert/strict'
import test from 'node:test'
import {
  bundleBudgets,
  collectStaticJavaScriptFiles,
  evaluateBudget,
  formatKib,
} from './check-bundle-budget.mjs'

test('collects an entry and its static JavaScript imports once', () => {
  const manifest = {
    'index.html': { file: 'assets/operations.js', imports: ['shared.js'], dynamicImports: ['lazy.js'] },
    'shared.js': { file: 'assets/shared.js', imports: ['nested.js'] },
    'nested.js': { file: 'assets/nested.js', imports: ['shared.js'] },
    'lazy.js': { file: 'assets/lazy.js' },
  }

  assert.deepEqual(collectStaticJavaScriptFiles(manifest, 'index.html'), [
    'assets/nested.js',
    'assets/operations.js',
    'assets/shared.js',
  ])
})

test('reports whether a compressed entry remains inside its budget', () => {
  const passing = evaluateBudget({
    name: 'Pilot',
    gzipBytes: bundleBudgets.pilot,
    budgetBytes: bundleBudgets.pilot,
    files: ['assets/pilot.js'],
  })
  assert.equal(passing.passed, true)
  assert.deepEqual(passing.files, ['assets/pilot.js'])
  assert.equal(evaluateBudget({ name: 'Pilot', gzipBytes: bundleBudgets.pilot + 1, budgetBytes: bundleBudgets.pilot }).passed, false)
  assert.equal(formatKib(2 * 1024), '2.00 KiB')
})
