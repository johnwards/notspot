import { test, expect } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://localhost:9108';

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

test.describe('Edit Columns', () => {
  test.beforeEach(async () => {
    await resetServer();
  });

  test('click Columns button opens dialog with property checkboxes', async ({ page }) => {
    await createContactViaAPI({ email: 'col@example.com', firstname: 'ColTest' });
    await page.goto(`${BASE_URL}/_ui/contacts`);
    await expect(page.locator('table tbody tr').first()).toBeVisible();

    // Click Columns button
    await page.getByTestId('edit-columns-btn').click();

    // Dialog should be visible
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByText('Edit Columns')).toBeVisible();

    // Should have checkboxes for properties
    await expect(page.getByTestId('column-checkbox-email')).toBeVisible();
    await expect(page.getByTestId('column-checkbox-firstname')).toBeVisible();
  });

  test('uncheck a column, save, verify it disappears from table', async ({ page }) => {
    await createContactViaAPI({ email: 'col@example.com', firstname: 'ColTest' });
    await page.goto(`${BASE_URL}/_ui/contacts`);
    await expect(page.locator('table tbody tr').first()).toBeVisible();

    // Verify "First Name" column exists
    const headers = page.locator('table thead th');
    await expect(headers.filter({ hasText: 'First Name' })).toBeVisible();

    // Open columns dialog
    await page.getByTestId('edit-columns-btn').click();
    await expect(page.getByRole('dialog')).toBeVisible();

    // Uncheck firstname
    await page.getByTestId('column-checkbox-firstname').click();

    // Save
    await page.getByTestId('save-columns-btn').click();

    // Dialog should close
    await expect(page.getByRole('dialog')).toBeHidden();

    // First Name column should be gone
    await expect(headers.filter({ hasText: 'First Name' })).toBeHidden();
  });

  test('Reset to Default restores original columns', async ({ page }) => {
    await createContactViaAPI({ email: 'col@example.com', firstname: 'ColTest' });
    await page.goto(`${BASE_URL}/_ui/contacts`);
    await expect(page.locator('table tbody tr').first()).toBeVisible();

    // First, customize columns (remove firstname)
    await page.getByTestId('edit-columns-btn').click();
    await page.getByTestId('column-checkbox-firstname').click();
    await page.getByTestId('save-columns-btn').click();

    // Verify firstname is gone
    const headers = page.locator('table thead th');
    await expect(headers.filter({ hasText: 'First Name' })).toBeHidden();

    // Now reset
    await page.getByTestId('edit-columns-btn').click();
    await page.getByTestId('reset-columns-btn').click();

    // First Name should be back
    await expect(headers.filter({ hasText: 'First Name' })).toBeVisible();
  });
});
