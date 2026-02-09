import { test, expect } from '@playwright/test';

const BASE = process.env.BASE_URL || 'http://localhost:9102';

// Helper: reset server data before tests that need clean state
async function resetServer() {
  await fetch(`${BASE}/_notspot/reset`, { method: 'POST' });
}

interface StageInfo {
  id: string;
  label: string;
  displayOrder: number;
}

interface PipelineInfo {
  id: string;
  label: string;
  stages: StageInfo[];
}

// Helper: fetch pipeline stages and return a map of label → stage ID
async function getDealPipeline(): Promise<PipelineInfo> {
  const res = await fetch(`${BASE}/crm/v3/pipelines/deals`);
  const data = await res.json();
  return data.results[0];
}

async function getTicketPipeline(): Promise<PipelineInfo> {
  const res = await fetch(`${BASE}/crm/v3/pipelines/tickets`);
  const data = await res.json();
  return data.results[0];
}

function stageIdByLabel(pipeline: PipelineInfo, label: string): string {
  const stage = pipeline.stages.find((s: StageInfo) => s.label === label);
  if (!stage) throw new Error(`Stage "${label}" not found in pipeline "${pipeline.label}"`);
  return stage.id;
}

// Helper: create a deal via API
async function createDeal(name: string, stageId: string, amount?: string) {
  const properties: Record<string, string> = {
    dealname: name,
    dealstage: stageId,
  };
  if (amount) properties.amount = amount;

  const res = await fetch(`${BASE}/crm/v3/objects/deals`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ properties }),
  });
  expect(res.ok).toBe(true);
  return res.json();
}

// Helper: create a ticket via API
async function createTicket(subject: string, stageId: string) {
  const res = await fetch(`${BASE}/crm/v3/objects/tickets`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      properties: { subject, hs_pipeline_stage: stageId },
    }),
  });
  expect(res.ok).toBe(true);
  return res.json();
}

// Helper: get a deal by ID via API
async function getDeal(id: string) {
  const res = await fetch(`${BASE}/crm/v3/objects/deals/${id}?properties=dealstage`);
  expect(res.ok).toBe(true);
  return res.json();
}

test.describe('Pipeline Board — Deals', () => {
  let pipeline: PipelineInfo;

  test.beforeEach(async () => {
    await resetServer();
    pipeline = await getDealPipeline();
  });

  test('board loads and shows stage columns', async ({ page }) => {
    await page.goto(`${BASE}/_ui/deals/board`);
    await page.waitForLoadState('networkidle');

    // Should display the pipeline label
    await expect(page.getByText('Sales Pipeline')).toBeVisible({ timeout: 10000 });

    // Should show all 7 deal stages as column headers
    await expect(page.getByText('Appointment Scheduled')).toBeVisible();
    await expect(page.getByText('Qualified To Buy')).toBeVisible();
    await expect(page.getByText('Presentation Scheduled')).toBeVisible();
    await expect(page.getByText('Decision Maker Bought-In')).toBeVisible();
    await expect(page.getByText('Contract Sent')).toBeVisible();
    await expect(page.getByText('Closed Won')).toBeVisible();
    await expect(page.getByText('Closed Lost')).toBeVisible();
  });

  test('deal card appears in correct stage column', async ({ page }) => {
    const stageId = stageIdByLabel(pipeline, 'Qualified To Buy');
    await createDeal('Test Deal Alpha', stageId);

    await page.goto(`${BASE}/_ui/deals/board`);
    await page.waitForLoadState('networkidle');

    // Wait for the board to load
    await expect(page.getByText('Sales Pipeline')).toBeVisible({ timeout: 10000 });

    // The deal card should be visible on the board
    await expect(page.getByText('Test Deal Alpha')).toBeVisible({ timeout: 10000 });

    // The deal should be in the "Qualified To Buy" column
    const columns = page.locator('.flex.gap-4 > div');
    const qualifiedColumn = columns.filter({ hasText: 'Qualified To Buy' });
    await expect(qualifiedColumn.getByText('Test Deal Alpha')).toBeVisible();
  });

  test('deal card shows deal name and amount', async ({ page }) => {
    const stageId = stageIdByLabel(pipeline, 'Appointment Scheduled');
    await createDeal('Big Enterprise Deal', stageId, '50000');

    await page.goto(`${BASE}/_ui/deals/board`);
    await page.waitForLoadState('networkidle');

    await expect(page.getByText('Sales Pipeline')).toBeVisible({ timeout: 10000 });

    // Card should show the deal name
    await expect(page.getByText('Big Enterprise Deal')).toBeVisible({ timeout: 10000 });

    // Card should show formatted amount ($50,000) — use .first() since it appears both on the card and column header
    await expect(page.getByText('$50,000').first()).toBeVisible();
  });

  test('multiple deals appear in their respective columns', async ({ page }) => {
    const stage1 = stageIdByLabel(pipeline, 'Appointment Scheduled');
    const stage3 = stageIdByLabel(pipeline, 'Presentation Scheduled');
    const stage5 = stageIdByLabel(pipeline, 'Contract Sent');

    await createDeal('Deal Stage 1', stage1);
    await createDeal('Deal Stage 3', stage3);
    await createDeal('Deal Stage 5', stage5);

    await page.goto(`${BASE}/_ui/deals/board`);
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('Sales Pipeline')).toBeVisible({ timeout: 10000 });

    // All deals should be visible
    await expect(page.getByText('Deal Stage 1')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Deal Stage 3')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Deal Stage 5')).toBeVisible({ timeout: 10000 });

    // Verify they are in the correct columns
    const columns = page.locator('.flex.gap-4 > div');

    const appointmentCol = columns.filter({ hasText: 'Appointment Scheduled' });
    await expect(appointmentCol.getByText('Deal Stage 1')).toBeVisible();

    const presentationCol = columns.filter({ hasText: 'Presentation Scheduled' });
    await expect(presentationCol.getByText('Deal Stage 3')).toBeVisible();

    const contractCol = columns.filter({ hasText: 'Contract Sent' });
    await expect(contractCol.getByText('Deal Stage 5')).toBeVisible();
  });

  test('item count shows correctly', async ({ page }) => {
    const stageId = stageIdByLabel(pipeline, 'Appointment Scheduled');
    await createDeal('Count Deal 1', stageId);
    await createDeal('Count Deal 2', stageId);

    await page.goto(`${BASE}/_ui/deals/board`);
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('Sales Pipeline')).toBeVisible({ timeout: 10000 });

    // The header should show "2 items"
    await expect(page.getByText('2 items')).toBeVisible({ timeout: 10000 });
  });

  test('empty board shows columns with no items', async ({ page }) => {
    await page.goto(`${BASE}/_ui/deals/board`);
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('Sales Pipeline')).toBeVisible({ timeout: 10000 });

    // Header should show "0 items"
    await expect(page.getByText('0 items')).toBeVisible();

    // Columns should show "No items" placeholder
    const noItemsLabels = page.getByText('No items');
    const count = await noItemsLabels.count();
    expect(count).toBe(7); // One per stage column
  });

  test('drag deal from one stage to another updates stage via API', async ({ page }) => {
    const sourceStageId = stageIdByLabel(pipeline, 'Appointment Scheduled');
    const targetStageLabel = 'Qualified To Buy';
    const targetStageId = stageIdByLabel(pipeline, targetStageLabel);

    const deal = await createDeal('Drag Me Deal', sourceStageId);
    const dealId = deal.id;

    await page.goto(`${BASE}/_ui/deals/board`);
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('Sales Pipeline')).toBeVisible({ timeout: 10000 });

    // Wait for the deal card to appear
    const card = page.getByText('Drag Me Deal');
    await expect(card).toBeVisible({ timeout: 10000 });

    // Find the target column droppable area
    const targetColumn = page.locator('.flex.gap-4 > div').filter({ hasText: targetStageLabel });
    const targetDrop = targetColumn.getByText('No items');
    await expect(targetDrop).toBeVisible();

    // dnd-kit uses PointerSensor, so we need to manually simulate pointer events
    // with sufficient movement (>5px activation constraint)
    const cardBox = await card.boundingBox();
    const targetBox = await targetDrop.boundingBox();
    expect(cardBox).not.toBeNull();
    expect(targetBox).not.toBeNull();

    const startX = cardBox!.x + cardBox!.width / 2;
    const startY = cardBox!.y + cardBox!.height / 2;
    const endX = targetBox!.x + targetBox!.width / 2;
    const endY = targetBox!.y + targetBox!.height / 2;

    await page.mouse.move(startX, startY);
    await page.mouse.down();
    // Move in steps to trigger the activation constraint (distance > 5px)
    await page.mouse.move(startX + 10, startY, { steps: 5 });
    await page.mouse.move(endX, endY, { steps: 10 });
    await page.mouse.up();

    // Wait for the API call to complete
    await page.waitForTimeout(1000);

    // Verify via API that the deal's stage was updated
    const updated = await getDeal(dealId);
    expect(updated.properties.dealstage).toBe(targetStageId);
  });
});

test.describe('Pipeline Board — Tickets', () => {
  let pipeline: PipelineInfo;

  test.beforeEach(async () => {
    await resetServer();
    pipeline = await getTicketPipeline();
  });

  test('ticket board loads with correct stages', async ({ page }) => {
    await page.goto(`${BASE}/_ui/tickets/board`);
    await page.waitForLoadState('networkidle');

    // Should display the pipeline label
    await expect(page.getByText('Support Pipeline')).toBeVisible({ timeout: 10000 });

    // Should show all 4 ticket pipeline stages
    await expect(page.getByRole('heading', { name: 'New' })).toBeVisible();
    await expect(page.getByText('Waiting on contact')).toBeVisible();
    await expect(page.getByText('Waiting on us')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Closed' })).toBeVisible();
  });

  test('ticket appears in correct stage column', async ({ page }) => {
    const stageId = stageIdByLabel(pipeline, 'Waiting on contact');
    await createTicket('Login Issue', stageId);

    await page.goto(`${BASE}/_ui/tickets/board`);
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('Support Pipeline')).toBeVisible({ timeout: 10000 });

    // The ticket should be visible
    await expect(page.getByText('Login Issue')).toBeVisible({ timeout: 10000 });

    // It should be in the "Waiting on contact" column
    const columns = page.locator('.flex.gap-4 > div');
    const waitingCol = columns.filter({ hasText: 'Waiting on contact' });
    await expect(waitingCol.getByText('Login Issue')).toBeVisible();
  });
});

test.describe('Pipeline Board — Navigation', () => {
  test('can navigate to board from sidebar or URL', async ({ page }) => {
    await page.goto(`${BASE}/_ui/deals/board`);
    await page.waitForLoadState('networkidle');

    // Board should load (not show a 404 or blank page)
    await expect(page.getByText('Sales Pipeline')).toBeVisible({ timeout: 10000 });

    // URL should be correct
    await expect(page).toHaveURL(/\/_ui\/deals\/board/);
  });
});
