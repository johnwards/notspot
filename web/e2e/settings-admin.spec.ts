import { test, expect } from '@playwright/test';

const BASE = process.env.BASE_URL || 'http://localhost:9103';

// Helper: reset server data before tests that need clean state
async function resetServer() {
  await fetch(`${BASE}/_notspot/reset`, { method: 'POST' });
}

test.describe('Sidebar Navigation', () => {
  test('sidebar shows CRM links and navigates correctly', async ({ page }) => {
    await page.goto(`${BASE}/_ui/contacts`);
    await page.waitForLoadState('networkidle');

    // Sidebar should show CRM object links
    const sidebar = page.locator('.flex.h-screen > div').first();
    await expect(sidebar.getByText('Contacts', { exact: true })).toBeVisible();
    await expect(sidebar.getByText('Companies', { exact: true })).toBeVisible();
    await expect(sidebar.getByText('Deals', { exact: true })).toBeVisible();
    await expect(sidebar.getByText('Tickets', { exact: true })).toBeVisible();

    // Click Companies link and verify navigation
    await sidebar.getByText('Companies', { exact: true }).click();
    await expect(page).toHaveURL(/\/_ui\/companies/);

    // Click Deals link and verify navigation
    await sidebar.getByText('Deals', { exact: true }).click();
    await expect(page).toHaveURL(/\/_ui\/deals/);

    // Click Tickets link and verify navigation
    await sidebar.getByText('Tickets', { exact: true }).click();
    await expect(page).toHaveURL(/\/_ui\/tickets/);

    // Click Contacts link and verify navigation
    await sidebar.getByText('Contacts', { exact: true }).click();
    await expect(page).toHaveURL(/\/_ui\/contacts/);
  });

  test('active link is highlighted', async ({ page }) => {
    await page.goto(`${BASE}/_ui/contacts`);
    await page.waitForLoadState('networkidle');

    // The Contacts link should have the active styling (bg-sidebar-accent)
    const contactsLink = page.locator('a[href*="contacts"]').first();
    await expect(contactsLink).toBeVisible();
    const contactsClass = await contactsLink.getAttribute('class');
    expect(contactsClass).toContain('bg-sidebar-accent');

    // Navigate to companies
    await page.locator('a[href*="companies"]').first().click();
    await page.waitForLoadState('networkidle');

    // Companies should now be active
    const companiesLink = page.locator('a[href*="companies"]').first();
    const companiesClass = await companiesLink.getAttribute('class');
    expect(companiesClass).toContain('bg-sidebar-accent');
  });

  test('sidebar shows settings and admin links', async ({ page }) => {
    await page.goto(`${BASE}/_ui/contacts`);
    await page.waitForLoadState('networkidle');

    const sidebar = page.locator('.flex.h-screen > div').first();
    await expect(sidebar.getByText('Properties')).toBeVisible();
    await expect(sidebar.getByText('Pipelines')).toBeVisible();
    await expect(sidebar.getByRole('link', { name: 'Admin' })).toBeVisible();
  });
});

test.describe('Settings > Properties', () => {
  test('properties page loads and shows property table', async ({ page }) => {
    await page.goto(`${BASE}/_ui/settings/properties`);
    await page.waitForLoadState('networkidle');

    // Page heading
    await expect(page.getByRole('heading', { name: 'Properties' })).toBeVisible();

    // Should show a table with property data (contacts is default)
    // Wait for data to load — look for table rows or known property names
    await expect(page.locator('table')).toBeVisible({ timeout: 10000 });

    // Should see at least one property (seeded data should have properties for contacts)
    const rows = page.locator('table tbody tr');
    await expect(rows.first()).toBeVisible({ timeout: 10000 });
  });

  test('can switch object type', async ({ page }) => {
    await page.goto(`${BASE}/_ui/settings/properties`);
    await page.waitForLoadState('networkidle');

    // Click the object type selector (default should be "Contacts")
    const trigger = page.locator('button[role="combobox"]');
    await trigger.click();

    // Select "Companies"
    await page.getByRole('option', { name: 'Companies' }).click();

    // Table should still be visible (may show different data)
    await expect(page.locator('table')).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Settings > Pipelines', () => {
  test('pipelines page loads and shows pipeline stages', async ({ page }) => {
    await page.goto(`${BASE}/_ui/settings/pipelines`);
    await page.waitForLoadState('networkidle');

    // Page heading
    await expect(page.getByRole('heading', { name: 'Pipelines' })).toBeVisible();

    // Wait for pipeline data to load — should show a pipeline card for deals (default)
    // Look for a table with stage data
    await expect(page.locator('table').first()).toBeVisible({ timeout: 10000 });

    // Should have stage labels like "Label" column header
    await expect(page.getByText('Label').first()).toBeVisible();
  });
});

test.describe('Admin Page', () => {
  test('admin page loads with reset button', async ({ page }) => {
    await page.goto(`${BASE}/_ui/admin`);
    await page.waitForLoadState('networkidle');

    // Page heading
    await expect(page.getByRole('heading', { name: 'Admin' })).toBeVisible();

    // Reset button should be visible
    await expect(page.getByRole('button', { name: /Reset Data/i })).toBeVisible();

    // Data Controls card should exist
    await expect(page.getByText('Data Controls')).toBeVisible();
  });

  test('admin reset workflow with confirmation', async ({ page }) => {
    // First create a contact
    const createRes = await fetch(`${BASE}/crm/v3/objects/contacts`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ properties: { email: 'reset-test@example.com', firstname: 'Reset', lastname: 'Test' } }),
    });
    expect(createRes.ok).toBe(true);
    const created = await createRes.json();
    const contactId = created.id;

    // Verify the contact exists
    const verifyRes = await fetch(`${BASE}/crm/v3/objects/contacts/${contactId}`);
    expect(verifyRes.ok).toBe(true);

    // Navigate to admin page
    await page.goto(`${BASE}/_ui/admin`);
    await page.waitForLoadState('networkidle');

    // Click reset button
    await page.getByRole('button', { name: /Reset Data/i }).click();

    // Confirmation dialog should appear
    await expect(page.getByText('This will delete all data')).toBeVisible();

    // Click confirm
    await page.getByRole('button', { name: /Reset Everything/i }).click();

    // Wait for success toast
    await expect(page.getByText('Data has been reset')).toBeVisible({ timeout: 10000 });

    // Verify the contact is gone
    const checkRes = await fetch(`${BASE}/crm/v3/objects/contacts/${contactId}`);
    expect(checkRes.ok).toBe(false);
  });
});

test.describe('Command Palette', () => {
  test('opens with Cmd+K and closes with Escape', async ({ page }) => {
    await page.goto(`${BASE}/_ui/contacts`);
    await page.waitForLoadState('networkidle');

    // Open command palette with Cmd+K
    await page.keyboard.press('Meta+k');

    // The dialog should open — look for the search input
    const searchInput = page.locator('[data-slot="command-input"]');
    await expect(searchInput).toBeVisible({ timeout: 5000 });

    // Close with Escape
    await page.keyboard.press('Escape');

    // Dialog should be gone
    await expect(searchInput).not.toBeVisible({ timeout: 5000 });
  });

  test('search returns results', async ({ page }) => {
    // Create a contact to search for
    await fetch(`${BASE}/crm/v3/objects/contacts`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ properties: { email: 'search-test@example.com', firstname: 'SearchTest', lastname: 'User' } }),
    });

    await page.goto(`${BASE}/_ui/contacts`);
    await page.waitForLoadState('networkidle');

    // Open command palette
    await page.keyboard.press('Meta+k');
    const searchInput = page.locator('[data-slot="command-input"]');
    await expect(searchInput).toBeVisible({ timeout: 5000 });

    // Type search query
    await searchInput.fill('SearchTest');

    // Wait for results (debounced 300ms + API call)
    await expect(page.locator('[data-slot="command-item"]').first()).toBeVisible({ timeout: 10000 });
  });

  test('search button in header opens palette', async ({ page }) => {
    await page.goto(`${BASE}/_ui/contacts`);
    await page.waitForLoadState('networkidle');

    // Click the search button in the header
    await page.getByRole('button', { name: /Search/i }).click();

    // Dialog should be open
    const searchInput = page.locator('[data-slot="command-input"]');
    await expect(searchInput).toBeVisible({ timeout: 5000 });
  });
});

test.describe('Route Fallback', () => {
  test('nonexistent route does not crash', async ({ page }) => {
    const response = await page.goto(`${BASE}/_ui/nonexistent-route-xyz`);

    // Should not get a server error
    expect(response?.status()).toBeLessThan(500);

    // Page should render something (not blank/crash)
    // The AppShell should still be present
    await expect(page.locator('.flex.h-screen')).toBeVisible({ timeout: 5000 });
  });
});
