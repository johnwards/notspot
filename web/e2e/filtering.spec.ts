import { test, expect } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://localhost:9103';

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

// Helper to find a saved view tab by its text content (not the dropdown trigger)
function savedViewTab(page: import('@playwright/test').Page, name: string) {
  return page.locator('[data-testid^="view-tab-view_"]', { hasText: name });
}

test.describe('Filtering and Saved Views', () => {
  test.beforeEach(async () => {
    await resetServer();
  });

  test('filter toggle button shows/hides filter panel', async ({ page }) => {
    await createContactViaAPI({ email: 'test@example.com', firstname: 'Test' });
    await page.goto(`${BASE_URL}/_ui/contacts`);

    // Wait for the table to load
    await expect(page.locator('table tbody tr').first()).toBeVisible();

    // Filter panel should not be visible initially
    await expect(page.getByTestId('filter-panel')).toBeHidden();

    // Click filter toggle button
    await page.getByTestId('filter-toggle').click();

    // Filter panel should be visible
    await expect(page.getByTestId('filter-panel')).toBeVisible();

    // Click again to hide
    await page.getByTestId('filter-toggle').click();

    // Filter panel should be hidden
    await expect(page.getByTestId('filter-panel')).toBeHidden();
  });

  test('add filter row shows property/operator/value controls', async ({ page }) => {
    await createContactViaAPI({ email: 'test@example.com', firstname: 'Test' });
    await page.goto(`${BASE_URL}/_ui/contacts`);
    await expect(page.locator('table tbody tr').first()).toBeVisible();

    // Open filter panel
    await page.getByTestId('filter-toggle').click();
    await expect(page.getByTestId('filter-panel')).toBeVisible();

    // Click "Add filter"
    await page.getByTestId('add-filter').click();

    // A filter row should appear with selects and an input
    const filterRow = page.getByTestId('filter-row');
    await expect(filterRow).toBeVisible();

    // Should have property select, operator select, and value input
    const selects = filterRow.locator('[data-slot="select-trigger"]');
    await expect(selects.first()).toBeVisible();
    await expect(selects.nth(1)).toBeVisible();
    await expect(filterRow.getByTestId('filter-value')).toBeVisible();
  });

  test('filter by email EQ returns matching contact only', async ({ page }) => {
    await createContactViaAPI({ email: 'alice@example.com', firstname: 'Alice' });
    await createContactViaAPI({ email: 'bob@example.com', firstname: 'Bob' });

    await page.goto(`${BASE_URL}/_ui/contacts`);
    await expect(page.getByText('alice@example.com')).toBeVisible();
    await expect(page.getByText('bob@example.com')).toBeVisible();

    // Open filter panel and add a filter
    await page.getByTestId('filter-toggle').click();
    await page.getByTestId('add-filter').click();

    const filterRow = page.getByTestId('filter-row');

    // Select "Email" property
    const propertyTrigger = filterRow.locator('[data-slot="select-trigger"]').first();
    await propertyTrigger.click();
    await page.getByRole('option', { name: 'Email' }).click();

    // Operator should default to "is equal to" (EQ)
    // Type value
    await filterRow.getByTestId('filter-value').fill('alice@example.com');

    // Wait for search to complete
    await page.waitForTimeout(500);

    // Alice should be visible, Bob should not
    await expect(page.getByText('alice@example.com')).toBeVisible();
    await expect(page.getByText('bob@example.com')).toBeHidden();
  });

  test('filter by HAS_PROPERTY on firstname shows results', async ({ page }) => {
    await createContactViaAPI({ email: 'has-name@example.com', firstname: 'WithName' });
    await createContactViaAPI({ email: 'no-name@example.com' });

    await page.goto(`${BASE_URL}/_ui/contacts`);

    // Wait for table to load (at least one row visible)
    await expect(page.locator('table tbody tr').first()).toBeVisible();

    // Open filter panel and add a filter
    await page.getByTestId('filter-toggle').click();
    await page.getByTestId('add-filter').click();

    const filterRow = page.getByTestId('filter-row');

    // Select "First Name" property
    const propertyTrigger = filterRow.locator('[data-slot="select-trigger"]').first();
    await propertyTrigger.click();
    await page.getByRole('option', { name: 'First Name' }).click();

    // Select "is known" (HAS_PROPERTY) operator
    const operatorTrigger = filterRow.locator('[data-slot="select-trigger"]').nth(1);
    await operatorTrigger.click();
    await page.getByRole('option', { name: 'is known' }).click();

    // Wait for search
    await page.waitForTimeout(500);

    // Contact with firstname should be visible
    await expect(page.getByText('has-name@example.com')).toBeVisible();
    // The value input should be hidden for HAS_PROPERTY
    await expect(filterRow.getByTestId('filter-value')).toBeHidden();
  });

  test('clear all filters restores full list', async ({ page }) => {
    await createContactViaAPI({ email: 'alice@example.com', firstname: 'Alice' });
    await createContactViaAPI({ email: 'bob@example.com', firstname: 'Bob' });

    await page.goto(`${BASE_URL}/_ui/contacts`);
    await expect(page.getByText('alice@example.com')).toBeVisible();
    await expect(page.getByText('bob@example.com')).toBeVisible();

    // Open filter panel and add a filter
    await page.getByTestId('filter-toggle').click();
    await page.getByTestId('add-filter').click();

    const filterRow = page.getByTestId('filter-row');

    // Select "Email" property and filter for alice
    const propertyTrigger = filterRow.locator('[data-slot="select-trigger"]').first();
    await propertyTrigger.click();
    await page.getByRole('option', { name: 'Email' }).click();
    await filterRow.getByTestId('filter-value').fill('alice@example.com');
    await page.waitForTimeout(500);

    // Only alice visible
    await expect(page.getByText('alice@example.com')).toBeVisible();
    await expect(page.getByText('bob@example.com')).toBeHidden();

    // Click "Clear all"
    await page.getByTestId('clear-all-filters').click();

    // Wait for the list to reload (filters cleared means useSearch=false, regular list loads)
    await page.waitForTimeout(500);

    // Both should be visible again
    await expect(page.getByText('alice@example.com')).toBeVisible();
    await expect(page.getByText('bob@example.com')).toBeVisible();
  });

  test('save current view creates a tab', async ({ page }) => {
    await createContactViaAPI({ email: 'alice@example.com', firstname: 'Alice' });
    await createContactViaAPI({ email: 'bob@example.com', firstname: 'Bob' });

    await page.goto(`${BASE_URL}/_ui/contacts`);
    // Clear any stale saved views from localStorage
    await page.evaluate(() => {
      Object.keys(localStorage)
        .filter((k) => k.startsWith('notspot_views_'))
        .forEach((k) => localStorage.removeItem(k));
    });
    await page.reload();
    await expect(page.getByText('alice@example.com')).toBeVisible();

    // Open filter panel and add a filter
    await page.getByTestId('filter-toggle').click();
    await page.getByTestId('add-filter').click();

    const filterRow = page.getByTestId('filter-row');
    const propertyTrigger = filterRow.locator('[data-slot="select-trigger"]').first();
    await propertyTrigger.click();
    await page.getByRole('option', { name: 'Email' }).click();
    await filterRow.getByTestId('filter-value').fill('alice@example.com');
    await page.waitForTimeout(500);

    // "Save view" button should appear
    await expect(page.getByTestId('save-view-button')).toBeVisible();
    await page.getByTestId('save-view-button').click();

    // Enter view name and save
    await page.getByTestId('save-view-name').fill('Alice Only');
    await page.getByTestId('save-view-confirm').click();

    // Tab should appear (use specific locator to avoid matching dropdown trigger)
    await expect(savedViewTab(page, 'Alice Only')).toBeVisible();
  });

  test('click saved view tab restores filters', async ({ page }) => {
    await createContactViaAPI({ email: 'alice@example.com', firstname: 'Alice' });
    await createContactViaAPI({ email: 'bob@example.com', firstname: 'Bob' });

    await page.goto(`${BASE_URL}/_ui/contacts`);
    await page.evaluate(() => {
      Object.keys(localStorage)
        .filter((k) => k.startsWith('notspot_views_'))
        .forEach((k) => localStorage.removeItem(k));
    });
    await page.reload();
    await expect(page.getByText('alice@example.com')).toBeVisible();

    // Open filter panel and add a filter
    await page.getByTestId('filter-toggle').click();
    await page.getByTestId('add-filter').click();

    const filterRow = page.getByTestId('filter-row');
    const propertyTrigger = filterRow.locator('[data-slot="select-trigger"]').first();
    await propertyTrigger.click();
    await page.getByRole('option', { name: 'Email' }).click();
    await filterRow.getByTestId('filter-value').fill('alice@example.com');
    await page.waitForTimeout(500);

    // Save the view
    await page.getByTestId('save-view-button').click();
    await page.getByTestId('save-view-name').fill('Alice View');
    await page.getByTestId('save-view-confirm').click();

    // Switch to "All contacts" tab
    await page.getByTestId('view-tab-all').click();
    await page.waitForTimeout(500);

    // Both should be visible (no filters)
    await expect(page.getByText('alice@example.com')).toBeVisible();
    await expect(page.getByText('bob@example.com')).toBeVisible();

    // Click the saved view tab
    await savedViewTab(page, 'Alice View').click();
    await page.waitForTimeout(500);

    // Filter panel should be visible with the filter applied
    await expect(page.getByTestId('filter-panel')).toBeVisible();
    // Only alice should be visible
    await expect(page.getByText('alice@example.com')).toBeVisible();
    await expect(page.getByText('bob@example.com')).toBeHidden();
  });

  test('delete saved view removes tab', async ({ page }) => {
    await createContactViaAPI({ email: 'alice@example.com', firstname: 'Alice' });

    await page.goto(`${BASE_URL}/_ui/contacts`);
    await page.evaluate(() => {
      Object.keys(localStorage)
        .filter((k) => k.startsWith('notspot_views_'))
        .forEach((k) => localStorage.removeItem(k));
    });
    await page.reload();
    await expect(page.getByText('alice@example.com')).toBeVisible();

    // Open filter panel and add a filter
    await page.getByTestId('filter-toggle').click();
    await page.getByTestId('add-filter').click();

    const filterRow = page.getByTestId('filter-row');
    const propertyTrigger = filterRow.locator('[data-slot="select-trigger"]').first();
    await propertyTrigger.click();
    await page.getByRole('option', { name: 'Email' }).click();
    await filterRow.getByTestId('filter-value').fill('alice@example.com');
    await page.waitForTimeout(500);

    // Save the view
    await page.getByTestId('save-view-button').click();
    await page.getByTestId('save-view-name').fill('Delete Me');
    await page.getByTestId('save-view-confirm').click();

    const viewTab = savedViewTab(page, 'Delete Me');
    await expect(viewTab).toBeVisible();

    // Hover over the group containing the view tab to reveal the options menu
    const viewGroup = viewTab.locator('..');
    await viewGroup.hover();
    const menuBtn = viewGroup.locator('[data-testid^="view-menu-"]');
    await menuBtn.click();

    // Click Delete
    await page.getByTestId('delete-view').click();

    // Tab should be gone
    await expect(savedViewTab(page, 'Delete Me')).toBeHidden();
  });

  test('saved views persist after page reload', async ({ page }) => {
    await createContactViaAPI({ email: 'alice@example.com', firstname: 'Alice' });

    await page.goto(`${BASE_URL}/_ui/contacts`);
    await page.evaluate(() => {
      Object.keys(localStorage)
        .filter((k) => k.startsWith('notspot_views_'))
        .forEach((k) => localStorage.removeItem(k));
    });
    await page.reload();
    await expect(page.getByText('alice@example.com')).toBeVisible();

    // Open filter panel and add a filter
    await page.getByTestId('filter-toggle').click();
    await page.getByTestId('add-filter').click();

    const filterRow = page.getByTestId('filter-row');
    const propertyTrigger = filterRow.locator('[data-slot="select-trigger"]').first();
    await propertyTrigger.click();
    await page.getByRole('option', { name: 'Email' }).click();
    await filterRow.getByTestId('filter-value').fill('alice@example.com');
    await page.waitForTimeout(500);

    // Save the view
    await page.getByTestId('save-view-button').click();
    await page.getByTestId('save-view-name').fill('Persistent View');
    await page.getByTestId('save-view-confirm').click();

    await expect(savedViewTab(page, 'Persistent View')).toBeVisible();

    // Reload page
    await page.reload();

    // Wait for page to load
    await expect(page.locator('table').first()).toBeVisible();

    // Saved view tab should still exist
    await expect(savedViewTab(page, 'Persistent View')).toBeVisible();
  });

  test('multiple AND filters narrow results correctly', async ({ page }) => {
    await createContactViaAPI({ email: 'alice@example.com', firstname: 'Alice', lastname: 'Smith' });
    await createContactViaAPI({ email: 'alice2@example.com', firstname: 'Alice', lastname: 'Jones' });
    await createContactViaAPI({ email: 'bob@example.com', firstname: 'Bob', lastname: 'Smith' });

    await page.goto(`${BASE_URL}/_ui/contacts`);
    await expect(page.getByText('alice@example.com')).toBeVisible();
    await expect(page.getByText('alice2@example.com')).toBeVisible();
    await expect(page.getByText('bob@example.com')).toBeVisible();

    // Open filter panel
    await page.getByTestId('filter-toggle').click();

    // Add first filter: firstname EQ Alice
    await page.getByTestId('add-filter').click();
    const firstFilter = page.getByTestId('filter-row').first();
    const firstPropTrigger = firstFilter.locator('[data-slot="select-trigger"]').first();
    await firstPropTrigger.click();
    await page.getByRole('option', { name: 'First Name' }).click();
    await firstFilter.getByTestId('filter-value').fill('Alice');

    // Add second filter: lastname EQ Smith
    await page.getByTestId('add-filter').click();
    const secondFilter = page.getByTestId('filter-row').nth(1);
    const secondPropTrigger = secondFilter.locator('[data-slot="select-trigger"]').first();
    await secondPropTrigger.click();
    await page.getByRole('option', { name: 'Last Name' }).click();
    await secondFilter.getByTestId('filter-value').fill('Smith');

    // Wait for search
    await page.waitForTimeout(500);

    // Only Alice Smith should be visible
    await expect(page.getByText('alice@example.com')).toBeVisible();
    await expect(page.getByText('alice2@example.com')).toBeHidden();
    await expect(page.getByText('bob@example.com')).toBeHidden();
  });
});
