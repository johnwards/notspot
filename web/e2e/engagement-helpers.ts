const ENGAGEMENT_TYPES = ['notes', 'calls', 'emails', 'tasks', 'meetings'] as const;

const CRM_TYPES = ['contacts', 'companies', 'deals'] as const;

interface PropertyDefinition {
  name: string;
  label: string;
  type: string;
  fieldType: string;
  groupName: string;
  options?: { label: string; value: string; displayOrder: number; hidden: boolean }[];
}

const ENGAGEMENT_PROPERTIES: Record<string, PropertyDefinition[]> = {
  notes: [
    {
      name: 'hs_note_body',
      label: 'Note Body',
      type: 'string',
      fieldType: 'textarea',
      groupName: 'engagement_info',
    },
  ],
  calls: [
    {
      name: 'hs_call_body',
      label: 'Call Body',
      type: 'string',
      fieldType: 'textarea',
      groupName: 'engagement_info',
    },
    {
      name: 'hs_call_direction',
      label: 'Call Direction',
      type: 'enumeration',
      fieldType: 'select',
      groupName: 'engagement_info',
      options: [
        { label: 'Inbound', value: 'INBOUND', displayOrder: 0, hidden: false },
        { label: 'Outbound', value: 'OUTBOUND', displayOrder: 1, hidden: false },
      ],
    },
    {
      name: 'hs_call_duration',
      label: 'Call Duration',
      type: 'number',
      fieldType: 'number',
      groupName: 'engagement_info',
    },
  ],
  emails: [
    {
      name: 'hs_email_subject',
      label: 'Email Subject',
      type: 'string',
      fieldType: 'text',
      groupName: 'engagement_info',
    },
    {
      name: 'hs_email_text',
      label: 'Email Text',
      type: 'string',
      fieldType: 'textarea',
      groupName: 'engagement_info',
    },
  ],
  tasks: [
    {
      name: 'hs_task_subject',
      label: 'Task Subject',
      type: 'string',
      fieldType: 'text',
      groupName: 'engagement_info',
    },
    {
      name: 'hs_task_body',
      label: 'Task Body',
      type: 'string',
      fieldType: 'textarea',
      groupName: 'engagement_info',
    },
    {
      name: 'hs_task_status',
      label: 'Task Status',
      type: 'enumeration',
      fieldType: 'select',
      groupName: 'engagement_info',
      options: [
        { label: 'Not Started', value: 'NOT_STARTED', displayOrder: 0, hidden: false },
        { label: 'In Progress', value: 'IN_PROGRESS', displayOrder: 1, hidden: false },
        { label: 'Completed', value: 'COMPLETED', displayOrder: 2, hidden: false },
      ],
    },
  ],
  meetings: [
    {
      name: 'hs_meeting_title',
      label: 'Meeting Title',
      type: 'string',
      fieldType: 'text',
      groupName: 'engagement_info',
    },
    {
      name: 'hs_meeting_start_time',
      label: 'Meeting Start Time',
      type: 'datetime',
      fieldType: 'text',
      groupName: 'engagement_info',
    },
    {
      name: 'hs_meeting_end_time',
      label: 'Meeting End Time',
      type: 'datetime',
      fieldType: 'text',
      groupName: 'engagement_info',
    },
  ],
};

export async function seedEngagementProperties(baseUrl: string): Promise<void> {
  for (const engagementType of ENGAGEMENT_TYPES) {
    const properties = ENGAGEMENT_PROPERTIES[engagementType];
    for (const prop of properties) {
      try {
        const resp = await fetch(`${baseUrl}/crm/v3/properties/${engagementType}`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(prop),
        });
        if (!resp.ok) {
          // Property might already exist on re-runs; ignore errors
        }
      } catch {
        // Ignore network errors
      }
    }
  }
}

export async function createAssociationTypes(baseUrl: string): Promise<void> {
  for (const engagementType of ENGAGEMENT_TYPES) {
    for (const crmType of CRM_TYPES) {
      try {
        const resp = await fetch(
          `${baseUrl}/crm/v4/associations/${engagementType}/${crmType}/labels`,
          {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              label: `engagement_to_${crmType.replace(/s$/, '')}`,
              name: `engagement_to_${crmType.replace(/s$/, '')}`,
            }),
          }
        );
        if (!resp.ok) {
          // Association type might already exist on re-runs; ignore errors
        }
      } catch {
        // Ignore network errors
      }
    }
  }
}

export async function createEngagement(
  baseUrl: string,
  type: string,
  props: Record<string, string>
): Promise<{ id: string }> {
  const resp = await fetch(`${baseUrl}/crm/v3/objects/${type}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ properties: props }),
  });
  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(`Failed to create ${type}: ${resp.status} ${text}`);
  }
  return resp.json();
}

export async function associateEngagement(
  baseUrl: string,
  engagementType: string,
  engagementId: string,
  objectType: string,
  objectId: string
): Promise<void> {
  const resp = await fetch(
    `${baseUrl}/crm/v4/objects/${engagementType}/${engagementId}/associations/default/${objectType}/${objectId}`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
    }
  );
  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(
      `Failed to associate ${engagementType} ${engagementId} to ${objectType} ${objectId}: ${resp.status} ${text}`
    );
  }
}

export async function seedFullEngagementData(
  baseUrl: string,
  contactId: string
): Promise<void> {
  const note = await createEngagement(baseUrl, 'notes', {
    hs_note_body: 'This is a test note body for the contact.',
  });
  await associateEngagement(baseUrl, 'notes', note.id, 'contacts', contactId);

  const call = await createEngagement(baseUrl, 'calls', {
    hs_call_body: 'Discussed project requirements.',
    hs_call_direction: 'INBOUND',
    hs_call_duration: '300',
  });
  await associateEngagement(baseUrl, 'calls', call.id, 'contacts', contactId);

  const email = await createEngagement(baseUrl, 'emails', {
    hs_email_subject: 'Follow-up on our meeting',
    hs_email_text: 'Thank you for your time today.',
  });
  await associateEngagement(baseUrl, 'emails', email.id, 'contacts', contactId);

  const task = await createEngagement(baseUrl, 'tasks', {
    hs_task_subject: 'Send proposal document',
    hs_task_body: 'Prepare and send the Q1 proposal.',
    hs_task_status: 'NOT_STARTED',
  });
  await associateEngagement(baseUrl, 'tasks', task.id, 'contacts', contactId);

  const meeting = await createEngagement(baseUrl, 'meetings', {
    hs_meeting_title: 'Quarterly review meeting',
    hs_meeting_start_time: '2025-06-15T10:00:00.000Z',
    hs_meeting_end_time: '2025-06-15T11:00:00.000Z',
  });
  await associateEngagement(baseUrl, 'meetings', meeting.id, 'contacts', contactId);
}
