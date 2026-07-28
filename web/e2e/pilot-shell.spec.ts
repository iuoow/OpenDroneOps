import { expect, test } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'

test.describe('Pilot Shell Browser Mock', () => {
  test.use({ viewport: { width: 390, height: 844 } })

  test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => window.localStorage.clear())
    await page.goto('/pilot.html')
    await expect(page.getByTestId('pilot-ready-shell')).toBeVisible()
  })

  test('field operator can navigate the compact read-only views with touch-sized controls', async ({ page }) => {
    const navigation = [
      ['device', 'pilot-view-device'],
      ['alerts', 'pilot-view-alerts'],
      ['more', 'pilot-view-more'],
      ['home', 'pilot-view-home'],
    ] as const

    for (const [destination, view] of navigation) {
      const control = page.getByTestId(`pilot-nav-${destination}`)
      await control.click()
      await expect(control).toHaveAttribute('aria-pressed', 'true')
      await expect(page.getByTestId(view)).toBeVisible()
    }

    const controlHeights = await page.locator('[data-testid^="pilot-nav-"]').evaluateAll((controls) =>
      controls.map((control) => Math.round(control.getBoundingClientRect().height)),
    )
    expect(controlHeights.every((height) => height >= 44)).toBe(true)
  })

  test('local draft and consent-first diagnostic flow remain explicit in Browser Mock mode', async ({ page }) => {
    await page.getByTestId('pilot-draft-body').fill('现场复核：等待风速下降后继续观察。')
    await page.getByTestId('pilot-save-draft').click()
    await expect(page.getByTestId('pilot-draft-list')).toBeVisible()

    await page.getByTestId('pilot-nav-more').click()
    await page.getByTestId('pilot-diagnostic-begin').click()
    await expect(page.getByTestId('pilot-diagnostic-accept')).toBeVisible()
    await page.getByTestId('pilot-diagnostic-accept').click()
    await expect(page.getByTestId('pilot-diagnostic-reset')).toBeVisible()
  })

  test('Browser Mock Pilot Shell has no serious axe violations', async ({ page }) => {
    const report = await new AxeBuilder({ page }).analyze()
    expect(report.violations.filter((violation) => ['critical', 'serious'].includes(violation.impact ?? ''))).toEqual([])
  })
})
