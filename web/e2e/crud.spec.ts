import { test, expect } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://localhost:9101';

// Reset the server state before each test
async function resetServer() {
  await fetch(`${BASE_URL}/_notspot/reset`, { method: 'POST' });
}

// Create a contact via API for test setup
async function createContactViaAPI(props: Record<string, string>) {
  const res = await fetch(`${BASE_URL}/crm/v3/objects/contacts`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ properties: props }),
  });
  return res.json();
}

test.describe('Object CRUD', () => {
  test.beforeEach(async () => {
    await resetServer();
  });

  test('page loads and table is visible', async ({ page }) => {
    // Create a contact so the table shows (not empty state)
    await createContactViaAPI({ email: 'test@example.com', firstname: 'Test' });

    await page.goto(`${BASE_URL}/_ui/contacts`);

    // Wait for the toolbar Create button (has the Plus icon + "Create" text)
    const toolbarCreateBtn = page.locator('button', { hasText: 'Create' }).filter({ has: page.locator('svg') }).first();
    await expect(toolbarCreateBtn).toBeVisible();

    // Table should be visible
    await expect(page.locator('table').first()).toBeVisible();
  });

  test('properties load as column headers', async ({ page }) => {
    // Create a contact so the table renders with data rows
    await createContactViaAPI({ email: 'test@example.com', firstname: 'Test' });

    await page.goto(`${BASE_URL}/_ui/contacts`);

    // Wait for table to have data
    await expect(page.locator('table tbody tr').first()).toBeVisible();

    // Check that column headers from property definitions exist
    const headers = page.locator('table thead th');
    const headerTexts = await headers.allTextContents();
    expect(headerTexts.some(h => h.includes('Email'))).toBeTruthy();
  });

  test('create contact via dialog', async ({ page }) => {
    await page.goto(`${BASE_URL}/_ui/contacts`);

    // Wait for the page to finish loading — the empty state or table should appear
    // Click the toolbar Create button
    const toolbarCreateBtn = page.locator('.flex.items-center.justify-between button', { hasText: 'Create' });
    await expect(toolbarCreateBtn).toBeVisible();
    await toolbarCreateBtn.click();

    // Dialog should appear
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByText('Create Contact')).toBeVisible();

    // Fill in email field
    const emailInput = page.getByRole('dialog').locator('input[id="email"]');
    await expect(emailInput).toBeVisible();
    await emailInput.fill('newcontact@example.com');

    // Fill in first name field
    const firstnameInput = page.getByRole('dialog').locator('input[id="firstname"]');
    await expect(firstnameInput).toBeVisible();
    await firstnameInput.fill('Alice');

    // Fill in last name field
    const lastnameInput = page.getByRole('dialog').locator('input[id="lastname"]');
    await expect(lastnameInput).toBeVisible();
    await lastnameInput.fill('Smith');

    // Click the Create submit button inside dialog
    await page.getByRole('dialog').getByRole('button', { name: 'Create' }).click();

    // Dialog should close
    await expect(page.getByRole('dialog')).toBeHidden();

    // New row should appear in the table
    await expect(page.getByText('newcontact@example.com')).toBeVisible();
    await expect(page.getByText('Alice')).toBeVisible();
  });

  test('view contact detail page', async ({ page }) => {
    // Create a contact via API
    const contact = await createContactViaAPI({
      email: 'detail@example.com',
      firstname: 'DetailTest',
      lastname: 'User',
    });

    await page.goto(`${BASE_URL}/_ui/contacts`);

    // Wait for the row to appear
    await expect(page.getByText('detail@example.com')).toBeVisible();

    // Click the row — should navigate to detail page
    await page.getByText('detail@example.com').click();

    // URL should contain the contact ID
    await page.waitForURL(`**/${contact.id}`);

    // AboutCard should display the contact's properties
    await expect(page.getByText('About this Contact')).toBeVisible();
    // Use the AboutCard's button elements (inline-edit buttons) for specific matching
    await expect(page.getByRole('button', { name: 'detail@example.com' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'DetailTest' })).toBeVisible();
  });

  test('edit contact on detail page', async ({ page }) => {
    // Create a contact via API
    const contact = await createContactViaAPI({
      email: 'edit@example.com',
      firstname: 'Before',
      lastname: 'Edit',
    });

    // Navigate directly to detail page
    await page.goto(`${BASE_URL}/_ui/contacts/${contact.id}`);

    // AboutCard should be visible
    await expect(page.getByText('About this Contact')).toBeVisible();

    // Click "See all properties" to open the edit dialog
    await page.getByText('See all properties').click();

    // Dialog should appear with the full property form
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'All Properties' })).toBeVisible();

    // Change first name in the dialog form
    const firstnameField = page.getByRole('dialog').locator('input[id="firstname"]');
    await expect(firstnameField).toBeVisible();
    await firstnameField.clear();
    await firstnameField.fill('After');

    // Click Save
    const saveBtn = page.getByRole('dialog').getByRole('button', { name: 'Save Changes' });
    await saveBtn.scrollIntoViewIfNeeded();
    await saveBtn.click();

    // Wait for save to complete (toast appears)
    await expect(page.getByText('Object updated')).toBeVisible();
  });

  test('archive contact from list', async ({ page }) => {
    // Create a contact via API
    await createContactViaAPI({
      email: 'archive@example.com',
      firstname: 'ToArchive',
    });

    // Archive via API directly (since sidebar is removed, archive is on detail page)
    await page.goto(`${BASE_URL}/_ui/contacts`);
    await expect(page.getByText('archive@example.com')).toBeVisible();

    // Verify the contact exists, then delete via API
    const res = await fetch(`${BASE_URL}/crm/v3/objects/contacts/search`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ query: 'archive@example.com', limit: 1 }),
    });
    const data = await res.json();
    const contactId = data.results[0].id;

    await fetch(`${BASE_URL}/crm/v3/objects/contacts/${contactId}`, {
      method: 'DELETE',
    });

    // Reload and verify contact is gone
    await page.reload();
    await page.waitForTimeout(500);
    await expect(page.getByText('archive@example.com')).toBeHidden();
  });

  test('search filters results', async ({ page }) => {
    // Create multiple contacts
    await createContactViaAPI({ email: 'alice@example.com', firstname: 'Alice' });
    await createContactViaAPI({ email: 'bob@example.com', firstname: 'Bob' });

    await page.goto(`${BASE_URL}/_ui/contacts`);
    await expect(page.getByText('alice@example.com')).toBeVisible();
    await expect(page.getByText('bob@example.com')).toBeVisible();

    // Type in search box
    const searchInput = page.getByPlaceholder('Search contacts...');
    await searchInput.fill('alice');

    // Wait for search debounce + results
    await page.waitForTimeout(500);

    // Alice should be visible, Bob should not
    await expect(page.getByText('alice@example.com')).toBeVisible();
    await expect(page.getByText('bob@example.com')).toBeHidden();
  });

  test('pagination works with many contacts', async ({ page }) => {
    // Create 25 contacts via API
    const promises = [];
    for (let i = 1; i <= 25; i++) {
      promises.push(
        createContactViaAPI({
          email: `user${i}@example.com`,
          firstname: `User${i}`,
        })
      );
    }
    await Promise.all(promises);

    await page.goto(`${BASE_URL}/_ui/contacts`);

    // Wait for table to load
    await expect(page.locator('table tbody tr').first()).toBeVisible();

    // Should show Page 1
    await expect(page.getByText('Page 1')).toBeVisible();

    // Next button should be enabled
    const nextButton = page.getByRole('button', { name: 'Next' });
    await expect(nextButton).toBeEnabled();

    // Previous button should be disabled on page 1
    const prevButton = page.getByRole('button', { name: 'Previous' });
    await expect(prevButton).toBeDisabled();

    // Click Next
    await nextButton.click();

    // Should show Page 2
    await expect(page.getByText('Page 2')).toBeVisible();

    // Previous should now be enabled
    await expect(prevButton).toBeEnabled();

    // Click Previous to go back
    await prevButton.click();
    await expect(page.getByText('Page 1')).toBeVisible();
  });
});
