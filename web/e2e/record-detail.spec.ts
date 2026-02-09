import { test, expect } from '@playwright/test';

const BASE_URL = process.env.BASE_URL || 'http://localhost:9101';

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

async function associateObjects(
  fromType: string,
  fromId: string,
  toType: string,
  toId: string,
) {
  await fetch(
    `${BASE_URL}/crm/v4/objects/${fromType}/${fromId}/associations/default/${toType}/${toId}`,
    { method: 'PUT', headers: { 'Content-Type': 'application/json' } },
  );
}

test.describe('Record Detail Page', () => {
  test.beforeEach(async () => {
    await resetServer();
  });

  test('row click in list navigates to detail page', async ({ page }) => {
    const contact = await createContactViaAPI({
      email: 'nav@example.com',
      firstname: 'NavTest',
    });

    await page.goto(`${BASE_URL}/_ui/contacts`);
    await expect(page.getByText('nav@example.com')).toBeVisible();

    await page.getByText('nav@example.com').click();

    // Should navigate to detail page URL
    await page.waitForURL(`**/${contact.id}`);
    expect(page.url()).toContain(`/contacts/${contact.id}`);
  });

  test('detail page shows three-panel layout', async ({ page }) => {
    const contact = await createContactViaAPI({
      email: 'layout@example.com',
      firstname: 'Layout',
      lastname: 'Test',
    });

    await page.goto(`${BASE_URL}/_ui/contacts/${contact.id}`);

    // Left panel: AboutCard
    await expect(page.getByText('About this Contact')).toBeVisible();

    // Middle panel: Overview tab
    await expect(page.getByRole('tab', { name: 'Overview' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Properties' })).toBeVisible();

    // Right panel: Associations
    await expect(page.getByRole('heading', { name: 'Associations' })).toBeVisible();
  });

  test('AboutCard displays key properties', async ({ page }) => {
    const contact = await createContactViaAPI({
      email: 'about@example.com',
      firstname: 'AboutTest',
      lastname: 'User',
    });

    await page.goto(`${BASE_URL}/_ui/contacts/${contact.id}`);

    await expect(page.getByText('About this Contact')).toBeVisible();
    // Use button role since AboutCard renders inline-edit buttons for values
    await expect(page.getByRole('button', { name: 'about@example.com' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'AboutTest' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'User' })).toBeVisible();
  });

  test('inline edit saves changes', async ({ page }) => {
    const contact = await createContactViaAPI({
      email: 'inline@example.com',
      firstname: 'BeforeEdit',
    });

    await page.goto(`${BASE_URL}/_ui/contacts/${contact.id}`);
    await expect(page.getByText('About this Contact')).toBeVisible();

    // Click on the firstname value to start inline editing (use button in AboutCard)
    await page.getByRole('button', { name: 'BeforeEdit' }).click();

    // Input should appear for editing
    const input = page.locator('input[id="firstname"]');
    await expect(input).toBeVisible();
    await input.clear();
    await input.fill('AfterEdit');
    await input.press('Enter');

    // Toast should confirm save
    await expect(page.getByText('Property updated')).toBeVisible();
  });

  test('See all properties opens dialog', async ({ page }) => {
    const contact = await createContactViaAPI({
      email: 'allprops@example.com',
      firstname: 'AllProps',
    });

    await page.goto(`${BASE_URL}/_ui/contacts/${contact.id}`);
    await expect(page.getByText('About this Contact')).toBeVisible();

    // Click "See all properties"
    await page.getByText('See all properties').click();

    // Dialog should appear
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'All Properties' })).toBeVisible();

    // Should have Save Changes button
    await expect(page.getByRole('dialog').getByRole('button', { name: 'Save Changes' })).toBeVisible();
  });

  test('association cards visible for related objects', async ({ page }) => {
    const contact = await createContactViaAPI({
      email: 'assoc@example.com',
      firstname: 'AssocTest',
    });
    const company = await createCompanyViaAPI({ name: 'TestCorp' });
    await associateObjects('contacts', contact.id, 'companies', company.id);

    await page.goto(`${BASE_URL}/_ui/contacts/${contact.id}`);

    // Right panel should show association cards
    await expect(page.getByRole('heading', { name: 'Associations' })).toBeVisible();
    // The association card button includes the count badge, e.g. "Companies 1"
    await expect(page.getByRole('button', { name: /Companies\s+1/ })).toBeVisible();
    await expect(page.getByText('TestCorp')).toBeVisible();
  });

  test('clicking association navigates to related record', async ({ page }) => {
    const contact = await createContactViaAPI({
      email: 'assocnav@example.com',
      firstname: 'AssocNav',
    });
    const company = await createCompanyViaAPI({ name: 'NavCorp' });
    await associateObjects('contacts', contact.id, 'companies', company.id);

    await page.goto(`${BASE_URL}/_ui/contacts/${contact.id}`);

    // Click on the associated company
    await expect(page.getByText('NavCorp')).toBeVisible();
    await page.getByText('NavCorp').click();

    // Should navigate to company detail page
    await page.waitForURL(`**/companies/${company.id}`);
    await expect(page.getByText('About this Companie')).toBeVisible();
  });

  test('breadcrumbs show correct path', async ({ page }) => {
    const contact = await createContactViaAPI({
      email: 'crumbs@example.com',
      firstname: 'CrumbTest',
    });

    await page.goto(`${BASE_URL}/_ui/contacts/${contact.id}`);

    // Wait for breadcrumbs to show - should have Notspot / Contacts / #id
    const nav = page.locator('header nav');
    await expect(nav.getByText('Notspot')).toBeVisible();
    await expect(nav.getByText('Contacts')).toBeVisible();
    // Breadcrumb shows #id format
    await expect(nav.getByText(`#${contact.id}`)).toBeVisible();
  });
});
