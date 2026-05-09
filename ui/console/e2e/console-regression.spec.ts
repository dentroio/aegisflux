import { test, expect } from '@playwright/test'

test.describe.configure({ timeout: 120_000 })

test.describe('authenticated console', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => {
      window.localStorage.setItem('aegisflux.labAuth', 'admin')
    })
  })

  async function assertNoSessionPlaceholder(page: import('@playwright/test').Page) {
    await expect(page.getByText('Checking session')).toHaveCount(0)
  }

  test('dashboard renders inside shell after lab auth', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByRole('heading', { name: 'Dashboard', exact: true })).toBeVisible({ timeout: 90_000 })
    await expect(page.getByTestId('console-sidebar-nav')).toBeVisible()
    await assertNoSessionPlaceholder(page)
  })

  test('/agents opens agents workbench (redirect or embedded shell)', async ({ page }) => {
    await page.goto('/agents')
    await page.waitForURL(/panel=agents|\/?$|\/agents$/, { timeout: 30_000 })
    await expect(page.getByText(/Agents Workbench|Agents/).first()).toBeVisible({ timeout: 90_000 })
    await expect(page.getByTestId('console-sidebar-nav')).toBeVisible()
    await assertNoSessionPlaceholder(page)
  })

  test('agents row navigates to agent detail when rows exist', async ({ page }) => {
    await page.goto('/?panel=agents')
    await page
      .getByRole('table')
      .or(page.getByText('No matching agents'))
      .first()
      .waitFor({ state: 'visible', timeout: 90_000 })
    const rows = page.locator('tbody tr')
    const n = await rows.count()
    if (n === 0) {
      test.skip()
      return
    }
    await Promise.all([page.waitForURL(/\/agents\/.+/), rows.first().click()])
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 30_000 })
    await assertNoSessionPlaceholder(page)
  })

  test('agent detail with auth leaves session gate', async ({ page }) => {
    await page.goto('/agents/e2e-smoke-device')
    await assertNoSessionPlaceholder(page)
    await expect(page.getByRole('heading', { level: 1 })).toContainText('e2e-smoke-device', { timeout: 90_000 })
  })

  test('/inventory opens inventory (redirect or embedded shell)', async ({ page }) => {
    await page.goto('/inventory')
    await page.waitForURL(/panel=inventory|\/?$|\/inventory$/, { timeout: 30_000 })
    await expect(page.getByText(/Inventory|Extension/i).first()).toBeVisible({ timeout: 90_000 })
    await expect(page.getByTestId('console-sidebar-nav')).toBeVisible()
    await assertNoSessionPlaceholder(page)
  })

  test('detections, controls, and events keep sidebar visible', async ({ page }) => {
    for (const path of ['/detections', '/control/controls', '/operate/events']) {
      await page.goto(path)
      await expect(page.getByTestId('console-sidebar-nav')).toBeVisible({ timeout: 90_000 })
      await expect(page.getByText('Checking session')).toHaveCount(0)
    }
  })

  test('route smoke: primary paths return OK', async ({ request }) => {
    for (const path of ['/', '/agents', '/inventory', '/detections', '/control/controls', '/operate/events']) {
      const res = await request.get(path)
      expect(res.ok(), `${path} -> ${res.status()}`).toBeTruthy()
    }
    const agentRes = await request.get('/agents/e2e-smoke-device')
    expect(agentRes.ok(), `agent detail -> ${agentRes.status()}`).toBeTruthy()
  })
})

test.describe('lab auth gate', () => {
  test('agent detail without auth redirects toward login', async ({ page }) => {
    await page.goto('/')
    await page.evaluate(() => window.localStorage.removeItem('aegisflux.labAuth'))
    await page.goto('/agents/e2e-smoke-device')
    await expect(page.getByRole('heading', { name: 'Sign in' })).toBeVisible({ timeout: 90_000 })
  })
})
