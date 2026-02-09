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

test.describe('Global Search', () => {
  test.beforeEach(async () => {
    await resetServer();
  });

  test('search result navigates to detail page', async ({ page }) => {
    const contact = await createContactViaAPI({
      email: 'search-test@example.com',
      firstname: 'SearchTest',
      lastname: 'User',
    });

    await page.goto(`${BASE_URL}/_ui/contacts`);
    await expect(page.locator('table').first()).toBeVisible();

    // Open command palette with Cmd+K
    await page.keyboard.press('Meta+k');
    await expect(page.getByPlaceholder('Search contacts, companies, deals, tickets...')).toBeVisible();

    // Type search query
    await page.getByPlaceholder('Search contacts, companies, deals, tickets...').fill('SearchTest');

    // Wait for results to appear in the command palette
    const dialog = page.locator('[role="dialog"]');
    await expect(dialog.getByText('search-test@example.com')).toBeVisible();

    // Click the result within the command palette
    await dialog.getByText('search-test@example.com').click();

    // Should navigate to the detail page
    await page.waitForURL(`**/contacts/${contact.id}`);
    expect(page.url()).toContain(`/contacts/${contact.id}`);
  });

  test('empty query shows no results', async ({ page }) => {
    await page.goto(`${BASE_URL}/_ui/contacts`);

    // Open command palette
    await page.keyboard.press('Meta+k');
    await expect(page.getByPlaceholder('Search contacts, companies, deals, tickets...')).toBeVisible();

    // With empty query, no result groups should appear
    await expect(page.locator('[cmdk-group]')).toHaveCount(0);
  });
});
