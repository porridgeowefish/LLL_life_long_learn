import { expect, test, type Page, type Route } from '@playwright/test';

async function json(route: Route, body: unknown, status = 200) {
  await route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
}

async function installStableBackend(page: Page) {
  await page.route('**/api/**', async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    if (path === '/api/projects' && request.method() === 'POST') {
      return json(route, { project: { id: 'e2e-project', slug: '浏览器验收', title: '浏览器验收', projectType: 'system-learning', status: 'active' } }, 201);
    }
    if (path === '/api/projects') return json(route, { projects: [] });
    if (path === '/api/health') return json(route, { status: 'ok', workspace: 'E2E', agentRuntime: { runtime: { name: 'fake', available: true } } });
    if (path === '/api/sessions/recent' || path === '/api/sessions/active') return json(route, { sessions: [] });
    if (path === '/api/activity') return json(route, { days: [], totalGrowth: 0 });
    if (path === '/api/folders') return json(route, { folders: [] });
    if (path === '/api/preferences') return json(route, { content: '' });
    return json(route, {});
  });
}

test('creates a system-learning project and enters its workspace', async ({ page }) => {
  await installStableBackend(page);
  await page.goto('/');
  await page.getByRole('button', { name: '新建项目' }).click();
  await page.getByText('系统学习', { exact: true }).click();
  await page.getByLabel('项目标题 *').fill('浏览器验收');
  await page.getByRole('button', { name: '创建项目' }).click();
  await expect(page).toHaveURL(/\/project\/%E6%B5%8F%E8%A7%88%E5%99%A8%E9%AA%8C%E6%94%B6(?:\/|$)/);
});

test('/memory keeps the compatibility redirect to preferences', async ({ page }) => {
  await installStableBackend(page);
  await page.goto('/memory');
  await expect(page).toHaveURL(/\/preferences$/);
  await expect(page.getByRole('heading', { name: '全局学习偏好' })).toBeVisible();
});
