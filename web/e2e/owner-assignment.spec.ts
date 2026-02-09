import { test, expect } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://localhost:9107';

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

async function getOwners() {
  const res = await fetch(`${BASE_URL}/crm/v3/owners`);
  return res.json();
}

test.describe('Owner Assignment', () => {
  test.beforeEach(async () => {
    await resetServer();
  });

  test('contact detail shows owner dropdown in AboutCard', async ({ page }) => {
    const contact = await createContactViaAPI({ email: 'owner-test@example.com', firstname: 'OwnerTest' });
    await page.goto(`${BASE_URL}/_ui/contacts/${contact.id}`);
    await expect(page.getByText('About this Contact')).toBeVisible();
    // Owner select should be visible
    await expect(page.getByTestId('owner-select')).toBeVisible();
  });

  test('select an owner, verify property saved', async ({ page }) => {
    const contact = await createContactViaAPI({ email: 'owner-save@example.com', firstname: 'Save' });
    const owners = await getOwners();
    const firstOwner = owners.results[0];

    await page.goto(`${BASE_URL}/_ui/contacts/${contact.id}`);
    await expect(page.getByTestId('owner-select')).toBeVisible();

    // Click the owner select
    await page.getByTestId('owner-select').click();
    // Select the first owner
    await page.getByRole('option', { name: `${firstOwner.firstName} ${firstOwner.lastName}` }).click();

    // Wait for save toast
    await expect(page.getByText('Property updated')).toBeVisible();
  });

  test('owner name appears in list view column', async ({ page }) => {
    const owners = await getOwners();
    const firstOwner = owners.results[0];

    // Create a contact with owner set
    await createContactViaAPI({
      email: 'owner-list@example.com',
      firstname: 'ListTest',
      hubspot_owner_id: firstOwner.id
    });

    await page.goto(`${BASE_URL}/_ui/contacts`);
    await expect(page.locator('table tbody tr').first()).toBeVisible();

    // Owner name should appear in the table (not raw ID)
    await expect(page.getByText(`${firstOwner.firstName} ${firstOwner.lastName}`)).toBeVisible();
  });
});
