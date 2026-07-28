import { expect, test } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'

test('operator can traverse the primary investigation paths in demo mode', async ({ page }) => {
  await page.goto('/app/demo/overview')
  await expect(page.getByRole('heading', { name: '实时态势' })).toBeVisible()
  await page.getByRole('link', { name: '设备管理' }).click()
  await expect(page.getByRole('heading', { name: '设备管理' })).toBeVisible()
  await page.getByRole('link', { name: '告警中心' }).click()
  await expect(page.getByRole('heading', { name: '告警中心' })).toBeVisible()
  await page.getByRole('link', { name: '指令中心' }).click()
  await expect(page.getByRole('heading', { name: '指令中心' })).toBeVisible()
  await page.getByRole('link', { name: '轨迹回放' }).click()
  await expect(page.getByRole('heading', { name: '轨迹回放' })).toBeVisible()
  await page.getByRole('link', { name: '系统运行' }).click()
  await expect(page.getByRole('heading', { name: '系统运行' })).toBeVisible()
})

test('desktop shell exposes keyboard skip navigation and has no serious axe violations', async ({ page }) => {
  await page.goto('/app/demo/overview')
  await page.keyboard.press('Tab')
  await expect(page.getByRole('link', { name: '跳至主内容' })).toBeFocused()
  const report = await new AxeBuilder({ page }).disableRules(['landmark-one-main']).analyze()
  expect(report.violations.filter((violation) => ['critical', 'serious'].includes(violation.impact ?? ''))).toEqual([])
})
