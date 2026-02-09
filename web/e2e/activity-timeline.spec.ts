import { test, expect } from '@playwright/test';
import { seedEngagementProperties, seedFullEngagementData } from './engagement-helpers';

const BASE_URL = process.env.BASE_URL || 'http://localhost:9102';

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

test.beforeAll(async () => {
  await seedEngagementProperties(BASE_URL);
});

test.describe('Activity Timeline', () => {
  let contactId: string;

  test.beforeEach(async () => {
    await resetServer();
    await seedEngagementProperties(BASE_URL);
    const contact = await createContactViaAPI({
      email: 'timeline@example.com',
      firstname: 'Timeline',
      lastname: 'Test',
    });
    contactId = contact.id;
    await seedFullEngagementData(BASE_URL, contactId);
  });

  test('Activity tab shows timeline with entries', async ({ page }) => {
    await page.goto(`${BASE_URL}/_ui/contacts/${contactId}`);
    await expect(page.getByText('About this Contact')).toBeVisible();

    // Click Activity tab
    await page.getByRole('tab', { name: 'Activity' }).click();

    // Timeline should show engagement entries
    const timeline = page.getByTestId('activity-timeline');
    await expect(timeline).toBeVisible();

    // Should have timeline items (at least one of each type we seeded)
    await expect(page.getByTestId('timeline-item-notes').first()).toBeVisible();
    await expect(page.getByTestId('timeline-item-calls').first()).toBeVisible();
    await expect(page.getByTestId('timeline-item-emails').first()).toBeVisible();
    await expect(page.getByTestId('timeline-item-tasks').first()).toBeVisible();
    await expect(page.getByTestId('timeline-item-meetings').first()).toBeVisible();
  });

  test('Timeline entries are grouped by type', async ({ page }) => {
    await page.goto(`${BASE_URL}/_ui/contacts/${contactId}`);
    await page.getByRole('tab', { name: 'Activity' }).click();

    const timeline = page.getByTestId('activity-timeline');
    await expect(timeline).toBeVisible();

    // All 5 engagement types should be present
    const items = page.locator('[data-testid^="timeline-item-"]');
    await expect(items).toHaveCount(5);
  });

  test('Note entry shows StickyNote icon area and body text', async ({ page }) => {
    await page.goto(`${BASE_URL}/_ui/contacts/${contactId}`);
    await page.getByRole('tab', { name: 'Activity' }).click();

    const noteItem = page.getByTestId('timeline-item-notes').first();
    await expect(noteItem).toBeVisible();

    // Icon area should be present
    await expect(page.getByTestId('timeline-icon-notes').first()).toBeVisible();

    // Note badge should show
    await expect(noteItem.locator('span', { hasText: 'Note' })).toBeVisible();

    // Body text from seed data
    await expect(noteItem.getByText('This is a test note body for the contact.')).toBeVisible();
  });

  test('Call entry shows Phone icon area', async ({ page }) => {
    await page.goto(`${BASE_URL}/_ui/contacts/${contactId}`);
    await page.getByRole('tab', { name: 'Activity' }).click();

    const callItem = page.getByTestId('timeline-item-calls').first();
    await expect(callItem).toBeVisible();

    // Icon area should be present
    await expect(page.getByTestId('timeline-icon-calls').first()).toBeVisible();

    // Call badge should show
    await expect(callItem.locator('span', { hasText: 'Call' })).toBeVisible();

    // Body text from seed data
    await expect(callItem.getByText('Discussed project requirements.')).toBeVisible();
  });

  test('Email entry shows Mail icon area and subject', async ({ page }) => {
    await page.goto(`${BASE_URL}/_ui/contacts/${contactId}`);
    await page.getByRole('tab', { name: 'Activity' }).click();

    const emailItem = page.getByTestId('timeline-item-emails').first();
    await expect(emailItem).toBeVisible();

    // Icon area should be present
    await expect(page.getByTestId('timeline-icon-emails').first()).toBeVisible();

    // Email badge should show
    await expect(emailItem.locator('span', { hasText: 'Email' })).toBeVisible();

    // Subject text from seed data
    await expect(emailItem.getByText(/Follow-up on our meeting/)).toBeVisible();
  });

  test('Action buttons visible in left panel (5 buttons)', async ({ page }) => {
    await page.goto(`${BASE_URL}/_ui/contacts/${contactId}`);
    await expect(page.getByText('About this Contact')).toBeVisible();

    // Action buttons container should be visible
    const actionButtons = page.getByTestId('action-buttons');
    await expect(actionButtons).toBeVisible();

    // Should have exactly 5 buttons
    await expect(page.getByTestId('action-btn-notes')).toBeVisible();
    await expect(page.getByTestId('action-btn-calls')).toBeVisible();
    await expect(page.getByTestId('action-btn-emails')).toBeVisible();
    await expect(page.getByTestId('action-btn-tasks')).toBeVisible();
    await expect(page.getByTestId('action-btn-meetings')).toBeVisible();
  });

  test('Click "Note" button opens create dialog', async ({ page }) => {
    await page.goto(`${BASE_URL}/_ui/contacts/${contactId}`);
    await expect(page.getByText('About this Contact')).toBeVisible();

    // Click the Note action button
    await page.getByTestId('action-btn-notes').click();

    // Dialog should appear
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Create Note' })).toBeVisible();

    // Note body textarea should be present
    await expect(page.locator('#hs_note_body')).toBeVisible();
  });

  test('Create a note via dialog - new entry appears in timeline', async ({ page }) => {
    await page.goto(`${BASE_URL}/_ui/contacts/${contactId}`);
    await expect(page.getByText('About this Contact')).toBeVisible();

    // Click Note action button
    await page.getByTestId('action-btn-notes').click();
    await expect(page.getByRole('dialog')).toBeVisible();

    // Fill in the note body
    await page.locator('#hs_note_body').fill('A brand new note created from UI');

    // Click Create button in dialog
    await page.getByRole('dialog').getByRole('button', { name: 'Create' }).click();

    // Dialog should close and toast should appear
    await expect(page.getByRole('dialog')).toBeHidden();
    await expect(page.getByText('Note created')).toBeVisible();

    // Switch to Activity tab
    await page.getByRole('tab', { name: 'Activity' }).click();

    // New note should appear in timeline
    await expect(page.getByText('A brand new note created from UI')).toBeVisible();
  });

  test('Create a call with direction - call appears', async ({ page }) => {
    await page.goto(`${BASE_URL}/_ui/contacts/${contactId}`);
    await expect(page.getByText('About this Contact')).toBeVisible();

    // Click Call action button
    await page.getByTestId('action-btn-calls').click();
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Create Call' })).toBeVisible();

    // Fill in call body
    await page.locator('#hs_call_body').fill('Follow-up call about the proposal');

    // Select direction
    await page.locator('#hs_call_direction').click();
    await page.getByRole('option', { name: 'Outbound' }).click();

    // Click Create
    await page.getByRole('dialog').getByRole('button', { name: 'Create' }).click();

    // Dialog should close and toast should appear
    await expect(page.getByRole('dialog')).toBeHidden();
    await expect(page.getByText('Call created')).toBeVisible();

    // Switch to Activity tab
    await page.getByRole('tab', { name: 'Activity' }).click();

    // New call should appear in timeline
    await expect(page.getByText('Follow-up call about the proposal')).toBeVisible();
  });

  test('Create a task - task appears with status', async ({ page }) => {
    await page.goto(`${BASE_URL}/_ui/contacts/${contactId}`);
    await expect(page.getByText('About this Contact')).toBeVisible();

    // Click Task action button
    await page.getByTestId('action-btn-tasks').click();
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Create Task' })).toBeVisible();

    // Fill in task subject
    await page.locator('#hs_task_subject').fill('Review quarterly report');

    // Fill in task body
    await page.locator('#hs_task_body').fill('Review and provide feedback on Q1 report');

    // Select status
    await page.locator('#hs_task_status').click();
    await page.getByRole('option', { name: 'Not Started' }).click();

    // Click Create
    await page.getByRole('dialog').getByRole('button', { name: 'Create' }).click();

    // Dialog should close and toast should appear
    await expect(page.getByRole('dialog')).toBeHidden();
    await expect(page.getByText('Task created')).toBeVisible();

    // Switch to Activity tab
    await page.getByRole('tab', { name: 'Activity' }).click();

    // New task should appear in timeline
    await expect(page.getByText(/Review quarterly report/)).toBeVisible();
  });
});
