import { test, expect } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://localhost:9104';

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

interface StageInfo {
  id: string;
  label: string;
  displayOrder: number;
  metadata: Record<string, string>;
}

interface PipelineInfo {
  id: string;
  label: string;
  stages: StageInfo[];
}

async function getDealPipeline(): Promise<PipelineInfo> {
  const res = await fetch(`${BASE_URL}/crm/v3/pipelines/deals`);
  const data = await res.json();
  return data.results[0];
}

function stageIdByLabel(pipeline: PipelineInfo, label: string): string {
  const stage = pipeline.stages.find((s: StageInfo) => s.label === label);
  if (!stage) throw new Error(`Stage "${label}" not found in pipeline "${pipeline.label}"`);
  return stage.id;
}

test.describe('Bulk Actions', () => {
  test.beforeEach(async () => {
    await resetServer();
  });

  test('selecting contacts shows bulk action bar with count', async ({ page }) => {
    const c1 = await createContactViaAPI({ email: 'bulk1@example.com', firstname: 'Bulk1' });
    await createContactViaAPI({ email: 'bulk2@example.com', firstname: 'Bulk2' });

    await page.goto(`${BASE_URL}/_ui/contacts`);
    await expect(page.getByText('bulk1@example.com')).toBeVisible();

    // Click the first row's checkbox
    await page.locator(`[data-testid="select-row-${c1.id}"]`).click();

    // Bulk bar should appear with "1 selected"
    const bar = page.locator('[data-testid="bulk-actions-bar"]');
    await expect(bar).toBeVisible();
    await expect(bar.getByText('1 selected')).toBeVisible();
  });

  test('header checkbox selects all visible rows', async ({ page }) => {
    await createContactViaAPI({ email: 'all1@example.com', firstname: 'All1' });
    await createContactViaAPI({ email: 'all2@example.com', firstname: 'All2' });
    await createContactViaAPI({ email: 'all3@example.com', firstname: 'All3' });

    await page.goto(`${BASE_URL}/_ui/contacts`);
    await expect(page.getByText('all1@example.com')).toBeVisible();

    // Click header select-all checkbox
    await page.locator('[data-testid="select-all-checkbox"]').click();

    // Bulk bar should show "3 selected"
    const bar = page.locator('[data-testid="bulk-actions-bar"]');
    await expect(bar).toBeVisible();
    await expect(bar.getByText('3 selected')).toBeVisible();
  });

  test('bulk delete archives selected contacts', async ({ page }) => {
    const c1 = await createContactViaAPI({ email: 'del1@example.com', firstname: 'Del1' });
    const c2 = await createContactViaAPI({ email: 'del2@example.com', firstname: 'Del2' });
    await createContactViaAPI({ email: 'keep@example.com', firstname: 'Keep' });

    await page.goto(`${BASE_URL}/_ui/contacts`);
    await expect(page.getByText('del1@example.com')).toBeVisible();

    // Select two contacts
    await page.locator(`[data-testid="select-row-${c1.id}"]`).click();
    await page.locator(`[data-testid="select-row-${c2.id}"]`).click();

    const bar = page.locator('[data-testid="bulk-actions-bar"]');
    await expect(bar.getByText('2 selected')).toBeVisible();

    // Click Delete
    await bar.locator('[data-testid="bulk-delete-btn"]').click();

    // Confirm in dialog
    await expect(page.getByRole('dialog')).toBeVisible();
    await page.locator('[data-testid="bulk-delete-confirm"]').click();

    // Wait for dialog to close (indicates deletion completed)
    await expect(page.getByRole('dialog')).toBeHidden({ timeout: 10000 });

    // Bulk bar should disappear (selection cleared after bulk op)
    await expect(bar).toBeHidden({ timeout: 10000 });

    // Deleted contacts should be gone, kept contact should remain
    await expect(page.getByText('del1@example.com')).toBeHidden({ timeout: 10000 });
    await expect(page.getByText('del2@example.com')).toBeHidden();
    await expect(page.getByText('keep@example.com')).toBeVisible();
  });

  test('bulk edit updates property on selected contacts', async ({ page }) => {
    const c1 = await createContactViaAPI({ email: 'edit1@example.com', firstname: 'Edit1' });
    const c2 = await createContactViaAPI({ email: 'edit2@example.com', firstname: 'Edit2' });

    await page.goto(`${BASE_URL}/_ui/contacts`);
    await expect(page.getByText('edit1@example.com')).toBeVisible();

    // Select both contacts
    await page.locator(`[data-testid="select-row-${c1.id}"]`).click();
    await page.locator(`[data-testid="select-row-${c2.id}"]`).click();

    const bar = page.locator('[data-testid="bulk-actions-bar"]');
    await expect(bar.getByText('2 selected')).toBeVisible();

    // Click Edit
    await bar.locator('[data-testid="bulk-edit-btn"]').click();

    // Dialog should appear
    await expect(page.getByRole('dialog')).toBeVisible();

    // Select "First Name" property
    await page.getByRole('dialog').locator('[data-slot="select-trigger"]').first().click();
    await page.getByRole('option', { name: 'First Name' }).click();

    // Enter a new value
    await page.locator('[data-testid="bulk-edit-value"]').fill('BulkUpdated');

    // Click Apply
    await page.locator('[data-testid="bulk-edit-apply"]').click();

    // Wait for update to complete
    await expect(page.getByText('Updated 2 contacts')).toBeVisible({ timeout: 10000 });

    // Verify via API that the property was updated
    const res1 = await fetch(`${BASE_URL}/crm/v3/objects/contacts/${c1.id}?properties=firstname`);
    const data1 = await res1.json();
    expect(data1.properties.firstname).toBe('BulkUpdated');

    const res2 = await fetch(`${BASE_URL}/crm/v3/objects/contacts/${c2.id}?properties=firstname`);
    const data2 = await res2.json();
    expect(data2.properties.firstname).toBe('BulkUpdated');
  });

  test('deselect all hides bulk bar', async ({ page }) => {
    const c1 = await createContactViaAPI({ email: 'desel@example.com', firstname: 'Desel' });

    await page.goto(`${BASE_URL}/_ui/contacts`);
    await expect(page.getByText('desel@example.com')).toBeVisible();

    // Select the contact
    await page.locator(`[data-testid="select-row-${c1.id}"]`).click();

    const bar = page.locator('[data-testid="bulk-actions-bar"]');
    await expect(bar).toBeVisible();

    // Click "Deselect all"
    await bar.locator('[data-testid="bulk-deselect-btn"]').click();

    // Bar should disappear
    await expect(bar).toBeHidden();
  });
});

test.describe('Lifecycle Stage Visualization', () => {
  test.beforeEach(async () => {
    await resetServer();
  });

  test('lifecycle bar shows on contact detail with correct stage highlighted', async ({ page }) => {
    const contact = await createContactViaAPI({
      email: 'lifecycle@example.com',
      firstname: 'Lifecycle',
      lifecyclestage: 'lead',
    });

    await page.goto(`${BASE_URL}/_ui/contacts/${contact.id}`);
    await expect(page.getByText('About this Contact')).toBeVisible();

    // Lifecycle bar should be visible
    const bar = page.locator('[data-testid="lifecycle-stage-bar"]');
    await expect(bar).toBeVisible();

    // Current stage marker should exist
    await expect(bar.locator('[data-testid="lifecycle-current-marker"]')).toBeVisible();

    // The "Lead" label should be styled as current (has font-semibold)
    const leadLabel = bar.locator('[data-testid="lifecycle-label-lead"]');
    await expect(leadLabel).toBeVisible();
    await expect(leadLabel).toHaveClass(/font-semibold/);
  });

  test('lifecycle bar does not appear on deal detail', async ({ page }) => {
    // Get deal pipeline to create a deal
    const pipeline = await getDealPipeline();
    const stageId = stageIdByLabel(pipeline, 'Appointment Scheduled');
    const deal = await createDealViaAPI({
      dealname: 'No Lifecycle Deal',
      dealstage: stageId,
    });

    await page.goto(`${BASE_URL}/_ui/deals/${deal.id}`);
    await expect(page.getByText('About this Deal')).toBeVisible();

    // Lifecycle bar should NOT be visible
    const bar = page.locator('[data-testid="lifecycle-stage-bar"]');
    await expect(bar).toHaveCount(0);
  });

  test('lifecycle progression: stages up to current are filled', async ({ page }) => {
    const contact = await createContactViaAPI({
      email: 'progression@example.com',
      firstname: 'Progress',
      lifecyclestage: 'opportunity',
    });

    await page.goto(`${BASE_URL}/_ui/contacts/${contact.id}`);
    await expect(page.getByText('About this Contact')).toBeVisible();

    const bar = page.locator('[data-testid="lifecycle-stage-bar"]');
    await expect(bar).toBeVisible();

    // Stages up to "opportunity" (index 4) should have data-filled
    // subscriber (0), lead (1), marketingqualifiedlead (2), salesqualifiedlead (3), opportunity (4)
    const filledSegments = bar.locator('[data-filled]');
    await expect(filledSegments).toHaveCount(5);

    // "Customer" and "Evangelist" should NOT be filled
    const customerSegment = bar.locator('[data-stage="customer"]');
    await expect(customerSegment).not.toHaveAttribute('data-filled');

    const evangelistSegment = bar.locator('[data-stage="evangelist"]');
    await expect(evangelistSegment).not.toHaveAttribute('data-filled');
  });
});

test.describe('Weighted Pipeline Totals', () => {
  test.beforeEach(async () => {
    await resetServer();
  });

  test('weighted total shows on deal board column', async ({ page }) => {
    const pipeline = await getDealPipeline();
    const stageId = stageIdByLabel(pipeline, 'Appointment Scheduled');

    // Create deals with amounts
    await createDealViaAPI({ dealname: 'Deal A', dealstage: stageId, amount: '100000' });
    await createDealViaAPI({ dealname: 'Deal B', dealstage: stageId, amount: '50000' });

    await page.goto(`${BASE_URL}/_ui/deals/board`);
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('Sales Pipeline')).toBeVisible({ timeout: 10000 });

    // Wait for deals to load
    await expect(page.getByText('Deal A')).toBeVisible({ timeout: 10000 });

    // The column should show a total
    const columns = page.locator('.flex.gap-4 > div');
    const appointmentCol = columns.filter({ hasText: 'Appointment Scheduled' });

    // Should show "Total: $150,000"
    const totalText = appointmentCol.locator('[data-testid="column-total"]');
    await expect(totalText).toContainText('$150,000');

    // Should show "Weighted:" line
    const weightedText = appointmentCol.locator('[data-testid="column-weighted"]');
    await expect(weightedText).toContainText('Weighted:');
  });
});

test.describe('Engagement Types in Sidebar', () => {
  test.beforeEach(async () => {
    await resetServer();
  });

  test('engagement types appear in sidebar navigation', async ({ page }) => {
    await page.goto(`${BASE_URL}/_ui/contacts`);

    // Engagements section should be visible
    const section = page.locator('[data-testid="engagements-section"]');
    await expect(section).toBeVisible();

    // All engagement types should be visible
    await expect(section.getByText('Notes')).toBeVisible();
    await expect(section.getByText('Calls')).toBeVisible();
    await expect(section.getByText('Emails')).toBeVisible();
    await expect(section.getByText('Tasks')).toBeVisible();
    await expect(section.getByText('Meetings')).toBeVisible();
  });

  test('clicking Notes in sidebar navigates to notes list view', async ({ page }) => {
    await page.goto(`${BASE_URL}/_ui/contacts`);

    // Wait for the sidebar engagements section to be visible
    const section = page.locator('[data-testid="engagements-section"]');
    await expect(section.getByText('Notes')).toBeVisible();

    // Click "Notes" in the sidebar
    await section.getByText('Notes').click();

    // Should navigate to /_ui/notes
    await page.waitForURL('**/_ui/notes');
    expect(page.url()).toContain('/_ui/notes');
  });
});
