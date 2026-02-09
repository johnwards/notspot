import { test, expect } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://localhost:9109';

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

async function createCompanyViaAPI(props: Record<string, string>) {
  const res = await fetch(`${BASE_URL}/crm/v3/objects/companies`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ properties: props }),
  });
  return res.json();
}

async function getAssociations(fromType: string, fromId: string, toType: string) {
  const res = await fetch(`${BASE_URL}/crm/v4/objects/${fromType}/${fromId}/associations/${toType}`);
  return res.json();
}

test.describe('Create with Association', () => {
  test.beforeEach(async () => {
    await resetServer();
  });

  test('create deal with associated contact', async ({ page }) => {
    // Create a contact to associate with
    const contact = await createContactViaAPI({
      email: 'assoc-deal@example.com',
      firstname: 'DealAssoc',
    });

    await page.goto(`${BASE_URL}/_ui/deals`);

    // Click Create button
    const createBtn = page.locator('.flex.items-center.justify-between button', { hasText: 'Create' });
    await expect(createBtn).toBeVisible();
    await createBtn.click();

    // Dialog should appear
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByText('Create Deal')).toBeVisible();

    // Fill in deal name
    const dealnameInput = page.getByRole('dialog').locator('input[id="dealname"]');
    await expect(dealnameInput).toBeVisible();
    await dealnameInput.fill('Test Deal');

    // Association section should be visible
    await expect(page.getByText('Associate with...')).toBeVisible();

    // Click Add for contacts
    await page.getByTestId('create-assoc-add-contacts').click();

    // Search dialog should open
    await page.getByPlaceholder('Search contacts...').fill('DealAssoc');

    // Wait for search results
    await page.waitForTimeout(500);

    // Select the contact
    await page.getByText('assoc-deal@example.com').click();

    // Back to create dialog, should see the pending association
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByText('assoc-deal@example.com')).toBeVisible();

    // Submit the form
    await page.getByRole('dialog').getByRole('button', { name: 'Create' }).click();

    // Wait for creation
    await expect(page.getByText(/created/i)).toBeVisible();

    // Verify association was created via API
    const dealsRes = await fetch(`${BASE_URL}/crm/v3/objects/deals/search`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        filterGroups: [{
          filters: [{
            propertyName: 'dealname',
            operator: 'EQ',
            value: 'Test Deal',
          }],
        }],
        limit: 1,
      }),
    });
    const dealsData = await dealsRes.json();
    expect(dealsData.results.length).toBeGreaterThan(0);

    const dealId = dealsData.results[0].id;
    const assocData = await getAssociations('deals', dealId, 'contacts');
    expect(assocData.results.length).toBeGreaterThan(0);
    expect(assocData.results[0].toObjectId).toBe(contact.id);
  });

  test('create contact with associated company', async ({ page }) => {
    const company = await createCompanyViaAPI({ name: 'AssocCorp' });

    await page.goto(`${BASE_URL}/_ui/contacts`);

    // Click Create
    const createBtn = page.locator('.flex.items-center.justify-between button', { hasText: 'Create' });
    await expect(createBtn).toBeVisible();
    await createBtn.click();

    await expect(page.getByRole('dialog')).toBeVisible();

    // Fill in email
    const emailInput = page.getByRole('dialog').locator('input[id="email"]');
    await expect(emailInput).toBeVisible();
    await emailInput.fill('with-company@example.com');

    // Should see associate section with companies
    await expect(page.getByText('Associate with...')).toBeVisible();
    await expect(page.getByTestId('create-assoc-add-companies')).toBeVisible();

    // No contacts association option (contacts can't associate with contacts)
    await expect(page.getByTestId('create-assoc-add-contacts')).toBeHidden();
  });

  test('association section does not appear for companies', async ({ page }) => {
    await page.goto(`${BASE_URL}/_ui/companies`);

    const createBtn = page.locator('.flex.items-center.justify-between button', { hasText: 'Create' });
    await expect(createBtn).toBeVisible();
    await createBtn.click();

    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByText('Create Company')).toBeVisible();

    // No association section
    await expect(page.getByText('Associate with...')).toBeHidden();
  });

  test('creating without associations still works', async ({ page }) => {
    await page.goto(`${BASE_URL}/_ui/deals`);

    const createBtn = page.locator('.flex.items-center.justify-between button', { hasText: 'Create' });
    await expect(createBtn).toBeVisible();
    await createBtn.click();

    await expect(page.getByRole('dialog')).toBeVisible();

    // Fill in deal name only
    const dealnameInput = page.getByRole('dialog').locator('input[id="dealname"]');
    await expect(dealnameInput).toBeVisible();
    await dealnameInput.fill('Solo Deal');

    // Submit without adding associations
    await page.getByRole('dialog').getByRole('button', { name: 'Create' }).click();

    // Should succeed
    await expect(page.getByText(/created/i)).toBeVisible();

    // Verify deal exists
    await expect(page.getByText('Solo Deal')).toBeVisible();
  });
});
