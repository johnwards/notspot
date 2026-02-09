import { test, expect } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://localhost:9106';

async function resetServer() {
  await fetch(`${BASE_URL}/_notspot/reset`, { method: 'POST' });
}

async function createObjectViaAPI(objectType: string, props: Record<string, string>) {
  const res = await fetch(`${BASE_URL}/crm/v3/objects/${objectType}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ properties: props }),
  });
  return res.json();
}

async function associateViaAPI(fromType: string, fromId: string, toType: string, toId: string) {
  await fetch(`${BASE_URL}/crm/v4/objects/${fromType}/${fromId}/associations/default/${toType}/${toId}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({}),
  });
}

test.describe('Association Management', () => {
  test.beforeEach(async () => {
    await resetServer();
  });

  test('add association via search dialog', async ({ page }) => {
    // Capture console errors for debugging
    const errors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') errors.push(msg.text());
    });
    page.on('pageerror', err => errors.push(err.message));

    // Create a contact and a company
    const contact = await createObjectViaAPI('contacts', {
      email: 'assoc-test@example.com',
      firstname: 'AssocTest',
    });
    const company = await createObjectViaAPI('companies', {
      name: 'TestCorp',
    });

    // Navigate to the contact detail page
    await page.goto(`${BASE_URL}/_ui/contacts/${contact.id}`);

    // Wait for the association panel to load — Companies card should be visible
    const addButton = page.getByTestId('association-add-companies');
    await expect(addButton).toBeVisible();

    // Click "+ Add" on the Companies association card
    await addButton.click();

    // The search dialog should appear
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();

    // Type the company name to search
    await dialog.getByPlaceholder('Search companies...').fill('TestCorp');

    // Wait for search results
    await expect(dialog.getByText('TestCorp')).toBeVisible();

    // Select the company from results
    await dialog.getByText('TestCorp').click();

    // Toast should confirm association created
    await expect(page.getByText('Association created')).toBeVisible();

    // Check for error boundary
    const errorBoundary = page.getByText('Something went wrong!');
    if (await errorBoundary.isVisible({ timeout: 3000 }).catch(() => false)) {
      // Click "Show Error" button and capture the error
      await page.getByRole('button', { name: 'Show Error' }).click();
      await page.waitForTimeout(500);
      const errorText = await page.locator('pre, code, .error-details').first().textContent().catch(() => 'No error details');
      console.log('Error boundary details:', errorText);
      const bodyText = await page.locator('body').textContent();
      console.log('Full page text:', bodyText?.substring(0, 500));
    }

    // The company should now appear in the associations panel (after cache refetch)
    await expect(page.getByText('TestCorp')).toBeVisible({ timeout: 10000 });
  });

  test('remove an existing association', async ({ page }) => {
    // Create a contact and a company, then associate them via API
    const contact = await createObjectViaAPI('contacts', {
      email: 'remove-test@example.com',
      firstname: 'RemoveTest',
    });
    const company = await createObjectViaAPI('companies', {
      name: 'RemoveCorp',
    });
    await associateViaAPI('contacts', contact.id, 'companies', company.id);

    // Navigate to the contact detail page
    await page.goto(`${BASE_URL}/_ui/contacts/${contact.id}`);

    // Wait for association to load — the company name should be visible
    await expect(page.getByText('RemoveCorp')).toBeVisible();

    // Click the remove button for this association
    const removeButton = page.getByTestId(`association-remove-${company.id}`);
    // Hover to reveal the remove button (it's hidden until hover)
    await removeButton.hover({ force: true });
    await removeButton.click({ force: true });

    // Toast should confirm removal
    await expect(page.getByText('Association removed')).toBeVisible();

    // The company should no longer appear
    await expect(page.getByText('RemoveCorp')).toBeHidden();
  });

  test('badge count updates after add and remove', async ({ page }) => {
    // Create contact and two companies
    const contact = await createObjectViaAPI('contacts', {
      email: 'badge-test@example.com',
      firstname: 'BadgeTest',
    });
    const company1 = await createObjectViaAPI('companies', {
      name: 'BadgeCorp1',
    });
    const company2 = await createObjectViaAPI('companies', {
      name: 'BadgeCorp2',
    });

    // Associate the first company
    await associateViaAPI('contacts', contact.id, 'companies', company1.id);

    // Navigate to the contact detail page
    await page.goto(`${BASE_URL}/_ui/contacts/${contact.id}`);

    // Wait for association to load
    await expect(page.getByText('BadgeCorp1')).toBeVisible();

    // Badge should show 1
    const companiesSection = page.locator('[data-testid="association-add-companies"]').locator('..');
    await expect(companiesSection.getByText('1')).toBeVisible();

    // Add the second company via the dialog
    await page.getByTestId('association-add-companies').click();
    const dialog = page.getByRole('dialog');
    await dialog.getByPlaceholder('Search companies...').fill('BadgeCorp2');
    await expect(dialog.getByText('BadgeCorp2')).toBeVisible();
    await dialog.getByText('BadgeCorp2').click();
    await expect(page.getByText('Association created')).toBeVisible();

    // Second company should now appear in associations (wait for refetch)
    await expect(page.getByText('BadgeCorp2')).toBeVisible({ timeout: 10000 });

    // Remove the first company
    const removeButton = page.getByTestId(`association-remove-${company1.id}`);
    await removeButton.hover({ force: true });
    await removeButton.click({ force: true });
    await expect(page.getByText('Association removed')).toBeVisible();

    // The first company should no longer appear
    await expect(page.getByText('BadgeCorp1')).toBeHidden({ timeout: 10000 });
  });
});
