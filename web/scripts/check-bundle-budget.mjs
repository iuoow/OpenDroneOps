import { readFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { gzipSync } from 'node:zlib'

const kib = 1024

export const bundleBudgets = Object.freeze({
  operations: 250 * kib,
  pilot: 200 * kib,
})

export function collectStaticJavaScriptFiles(manifest, entryKey) {
  const visited = new Set()
  const files = new Set()

  function visit(key) {
    if (visited.has(key)) return
    visited.add(key)
    const entry = manifest[key]
    if (!entry) throw new Error(`BUNDLE_MANIFEST_ENTRY_MISSING: ${key}`)
    if (entry.file?.endsWith('.js')) files.add(entry.file)
    for (const dependency of entry.imports ?? []) visit(dependency)
  }

  visit(entryKey)
  return [...files].sort()
}

export function formatKib(bytes) {
  return `${(bytes / kib).toFixed(2)} KiB`
}

export function evaluateBudget({ name, gzipBytes, budgetBytes, files = [] }) {
  return {
    name,
    gzipBytes,
    budgetBytes,
    files,
    passed: gzipBytes <= budgetBytes,
  }
}

export async function measureEntry({ manifest, entryKey, distDirectory }) {
  const files = collectStaticJavaScriptFiles(manifest, entryKey)
  const chunks = await Promise.all(files.map((file) => readFile(path.join(distDirectory, file))))
  return {
    files,
    gzipBytes: chunks.reduce((total, chunk) => total + gzipSync(chunk).byteLength, 0),
  }
}

export async function checkBundleBudgets({ distDirectory }) {
  const manifest = JSON.parse(await readFile(path.join(distDirectory, '.vite', 'manifest.json'), 'utf8'))
  const checks = await Promise.all([
    measureEntry({ manifest, entryKey: 'index.html', distDirectory }).then((measurement) =>
      evaluateBudget({ name: 'Operations initial JavaScript', budgetBytes: bundleBudgets.operations, ...measurement }),
    ),
    measureEntry({ manifest, entryKey: 'pilot.html', distDirectory }).then((measurement) =>
      evaluateBudget({ name: 'Pilot initial JavaScript', budgetBytes: bundleBudgets.pilot, ...measurement }),
    ),
  ])

  return checks
}

async function main() {
  const currentDirectory = path.dirname(fileURLToPath(import.meta.url))
  const distDirectory = path.resolve(currentDirectory, '..', 'dist')
  const checks = await checkBundleBudgets({ distDirectory })

  for (const check of checks) {
    const result = check.passed ? 'PASS' : 'FAIL'
    console.log(`${result} ${check.name}: ${formatKib(check.gzipBytes)} gzip / ${formatKib(check.budgetBytes)} limit`)
    console.log(`  static chunks: ${check.files.join(', ')}`)
  }

  if (checks.some((check) => !check.passed)) {
    process.exitCode = 1
  }
}

if (process.argv[1] && fileURLToPath(import.meta.url) === path.resolve(process.argv[1])) {
  main().catch((error) => {
    console.error(error instanceof Error ? error.message : error)
    process.exitCode = 1
  })
}
