import { test, expect } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://localhost:9105';

async function resetServer() {
  await fetch(`${BASE_URL}/_notspot/reset`, { method: 'POST' });
}

async function createContactViaAPI(props: Record<string, string>) {
  const res = await fetch(`${BASE_URL}/crm/v3/objects/contacts`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ properties: props }),
  });
  return res.json();
}

async function createDealViaAPI(props: Record<string, string>) {
  const res = await fetch(`${BASE_URL}/crm/v3/objects/deals`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ properties: props }),
  });
  return res.json();
}

test.describe('Dashboard', () => {
  test.beforeEach(async () => {
    await resetServer();
  });

  test('dashboard loads with widget sections visible', async ({ page }) => {
    await page.goto(`${BASE_URL}/_ui/dashboard`);

    await expect(page.getByText('Dashboard')).toBeVisible();
    await expect(page.getByTestId('recent-records')).toBeVisible();
    await expect(page.getByTestId('deal-pipeline-summary')).toBeVisible();
    await expect(page.getByTestId('tasks-summary')).toBeVisible();
  });

  test('recent records shows created contacts and deals', async ({ page }) => {
    await createContactViaAPI({ email: 'dash@example.com', firstname: 'DashUser' });
    await createDealViaAPI({ dealname: 'Big Deal', amount: '5000' });

    await page.goto(`${BASE_URL}/_ui/dashboard`);

    const recentRecords = page.getByTestId('recent-records');
    await expect(recentRecords.getByText('dash@example.com')).toBeVisible();
    await expect(recentRecords.getByText('Big Deal')).toBeVisible();
  });

  test('click recent record navigates to detail page', async ({ page }) => {
    const contact = await createContactViaAPI({ email: 'click@example.com', firstname: 'ClickMe' });

    await page.goto(`${BASE_URL}/_ui/dashboard`);

    const recentRecords = page.getByTestId('recent-records');
    await expect(recentRecords.getByText('click@example.com')).toBeVisible();

    await recentRecords.getByText('click@example.com').click();

    await page.waitForURL(`**/contacts/${contact.id}`);
    expect(page.url()).toContain(`/contacts/${contact.id}`);
  });

  test('root /_ui/ loads dashboard', async ({ page }) => {
    await page.goto(`${BASE_URL}/_ui/`);

    // Should redirect to dashboard
    await page.waitForURL('**/dashboard');
    await expect(page.getByText('Dashboard')).toBeVisible();
  });
});
