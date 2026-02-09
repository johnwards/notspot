#!/usr/bin/env npx tsx
/**
 * Seed data script for Notspot HubSpot mock server.
 *
 * Generates ~950 realistic B2B SaaS CRM records using deterministic data pools.
 * Companies, contacts, deals, tickets, and engagements with full associations.
 *
 * Usage:
 *   npx tsx scripts/seed-data.ts [BASE_URL]
 *
 * BASE_URL defaults to http://localhost:8080
 */

const BASE_URL = (process.argv[2] || "http://localhost:8080").replace(/\/+$/, "");

// ---------------------------------------------------------------------------
// Data Pools
// ---------------------------------------------------------------------------

const FIRST_NAMES = [
  "Sarah", "David", "Priya", "Tom", "Mike", "Linda", "James", "Aisha",
  "Carlos", "Rachel", "Gregory", "Emily", "Robert", "Jessica", "Derek",
  "Natalie", "Anthony", "Samantha", "Marcus", "Diana", "Kevin", "Laura",
  "Jason", "Olivia", "Brian", "Michelle", "Ethan", "Hannah", "Victor",
  "Sophie", "Alex", "Maria", "Ryan", "Nicole", "Daniel", "Christina",
  "Jordan", "Fatima", "Nathan", "Yuki", "Omar", "Elena", "Tyler", "Mei",
  "Hassan", "Clara", "Leo", "Simone", "Andre", "Ingrid", "Raj", "Zoe",
  "Patrick", "Leila", "Kofi", "Anastasia", "Mateo", "Freya", "Isaiah", "Nadia",
];

const LAST_NAMES = [
  "Chen", "Park", "Sharma", "Raines", "O'Brien", "Torres", "Whitfield",
  "Rahman", "Mendez", "Kim", "Lawson", "Zhang", "Singh", "Huang", "Okafor",
  "Reed", "Russo", "Lee", "Brown", "Petrova", "Nakamura", "Bennett",
  "Morales", "Foster", "Walsh", "Gupta", "Clarke", "Johansson", "Adeyemi",
  "Martinez", "Bergstrom", "Tanaka", "Mueller", "Costa", "Ivanova", "Patel",
  "Kowalski", "Dubois", "Andersen", "Watanabe", "Al-Hassan", "Reyes",
  "Volkov", "Chandra", "Eriksson", "Nwankwo", "Delgado", "Hoffmann",
  "Suzuki", "Ferreira", "Lindgren", "Ota", "Svensson", "Bouchard",
  "Takahashi", "Pereira", "Jansen", "Rossi", "Fischer", "Novak",
];

const COMPANY_PREFIXES = [
  "Meridian", "Apex", "Pinnacle", "Cascade", "Northstar", "Bluewave",
  "Ironforge", "TerraVolt", "Silverline", "Verdant", "Quantum", "Horizon",
  "Nexus", "Vanguard", "Zenith", "Stellar", "Prism", "Catalyst",
  "Summit", "Ember", "Nova", "Atlas", "Kinetic", "Luminary", "Helix",
  "Stratos", "Fusion", "Cobalt", "Vertex", "Aether", "Onyx", "Zephyr",
  "Radiant", "Titan", "Eclipse", "Quartz", "Axiom", "Polaris", "Cipher",
  "Mosaic",
];

const COMPANY_SUFFIXES = [
  "Solutions", "Group", "Labs", "Technologies", "Inc", "Systems",
  "Analytics", "Dynamics", "Partners", "Digital", "Cloud", "Software",
  "Consulting", "Industries", "Ventures", "Networks", "Global", "Services",
  "Platforms", "AI",
];

const DOMAIN_TLDS = [".io", ".com", ".co", ".tech", ".dev", ".ai", ".cloud"];

const INDUSTRIES = [
  "COMPUTER_SOFTWARE", "MANUFACTURING", "HOSPITAL_HEALTH_CARE",
  "FINANCIAL_SERVICES", "LOGISTICS_SUPPLY_CHAIN", "EDUCATION",
  "MANAGEMENT_CONSULTING", "MARKETING_ADVERTISING", "RETAIL",
  "RENEWABLES_ENVIRONMENT", "TELECOMMUNICATIONS", "REAL_ESTATE",
  "CONSTRUCTION", "FOOD_BEVERAGE", "AUTOMOTIVE", "BIOTECHNOLOGY",
  "GOVERNMENT", "INSURANCE", "MEDIA", "PROFESSIONAL_SERVICES",
];

const PHONE_AREA_CODES = [
  "415", "312", "617", "212", "206", "303", "512", "503", "469", "720",
  "650", "310", "404", "617", "919", "858", "480", "971", "704", "305",
];

const NOTE_TEMPLATES = [
  "Spoke with {contact} about the Q2 renewal. They confirmed they want to expand to 50 seats. Will send updated proposal by Friday.",
  "{contact} mentioned the team is evaluating competitor products. Need to schedule a demo of our new features ASAP.",
  "Confirmed the rollout at {company} went smoothly. Team adoption at 85% after two weeks.",
  "{contact} flagged a concern about compliance for their new use case. Looped in our security team for review.",
  "{contact} is the final decision maker. They're reviewing the contract with legal this week.",
  "Initial discovery call with {contact} went well. {company} is looking for analytics across multiple accounts.",
  "{contact} is excited about the demo. Wants to bring their CTO to the next call.",
  "Discussed integration timeline with {contact}. {company} needs the API ready by end of quarter.",
  "{contact} requested a custom pricing proposal for {company}. Preparing tiered options.",
  "Follow-up from the conference. {contact} from {company} expressed strong interest in our platform.",
  "Quarterly check-in with {contact}. Everything running smoothly. Discussed upcoming feature requests.",
  "{contact} raised a support escalation. Resolved within 2 hours. Customer satisfied with response time.",
  "Onboarding session with {contact} completed. Covered dashboard setup and user permissions.",
  "Strategy session with {contact} about {company}'s growth plans. Potential for 3x expansion next year.",
  "{contact} mentioned they're presenting our ROI to the board next month. Sent case studies to support.",
  "Product feedback from {contact}: wants better reporting granularity and export options.",
  "Training session delivered to {company}'s team. 12 attendees, positive feedback overall.",
  "{contact} confirmed budget approval for the next phase. Moving to contract review.",
  "Discussed data migration timeline with {contact}. Estimated 2-3 weeks for full import.",
  "Renewal conversation with {contact}. They want to add 3 new modules to their subscription.",
  "{contact} requested a reference call with another customer in their industry.",
  "Technical deep-dive with {contact}'s engineering team. Covered API architecture and webhooks.",
  "Executive briefing with {contact}. Aligned on strategic roadmap for the next 12 months.",
  "Post-implementation review with {contact}. All KPIs met, moving to optimization phase.",
  "{contact} shared that {company} saved 40% on operational costs since adopting our platform.",
];

const CALL_SUMMARIES = [
  "Quarterly business review. Discussed roadmap alignment and upcoming feature requests. Happy with support.",
  "Called about a permissions issue affecting team members. Escalated to engineering.",
  "Follow-up on expansion deal. Aligning budget with VP for approval.",
  "Discussed compliance module timeline. Needs it live before their audit.",
  "Cold call — interested in a product demo. Scheduling for next week.",
  "Had questions about uptime SLA. Sent follow-up documentation.",
  "Reviewed implementation progress. On track for go-live next month.",
  "Pricing discussion for enterprise tier. Sending formal proposal.",
  "Technical support call. Resolved API integration issue with webhooks.",
  "Check-in call. Everything running smoothly, discussed training needs.",
  "Onboarding kickoff call. Reviewed timeline and assigned project leads.",
  "Discussed renewal terms. Wants multi-year discount options.",
  "Inbound support request. Database sync issue resolved during call.",
  "Demo of new analytics dashboard. Very positive reception from the team.",
  "Follow-up on outstanding contract redlines from legal review.",
  "Discussed data residency requirements for EU compliance.",
  "Partnership exploration call. Potential co-selling opportunity.",
  "Escalation call about downtime incident. Provided root cause analysis.",
  "Quarterly planning session. Mapped out feature priorities for next quarter.",
  "Migration planning call. Reviewed data mapping and cutover strategy.",
];

const EMAIL_SUBJECTS = [
  "Re: Enterprise Annual Renewal",
  "Migration Plan Update",
  "Contract for Review",
  "Pricing Update — Enterprise Tier",
  "Demo Scheduling",
  "Welcome to the Platform!",
  "Re: Feature Request — Dark Mode",
  "Quarterly Business Review Recap",
  "Implementation Timeline Update",
  "Invoice Correction — {company}",
  "Training Session Follow-up",
  "Product Roadmap Preview",
  "Security Compliance Documentation",
  "Custom Report Delivery",
  "Integration Guide & API Keys",
  "Renewal Proposal — {company}",
  "Meeting Notes — Strategy Session",
  "Support Ticket Resolution",
  "Onboarding Checklist",
  "Partnership Opportunity",
];

const EMAIL_BODIES = [
  "Please find the updated proposal attached with the expanded seat count. Let me know if you have any questions.",
  "Here is the detailed migration plan. We estimate 2-3 weeks for the full import of historical data.",
  "Attached is the final contract. Please review with your legal team and let us know if there are any redlines.",
  "Great news — we can offer the add-on at a competitive rate which includes premium support.",
  "I'd love to schedule a demo for your team. Do you have availability next Tuesday or Thursday?",
  "Welcome aboard! Here's your getting started guide and links to our onboarding resources.",
  "Thanks for the feature request. It's on our Q3 roadmap. I'll keep you posted on progress.",
  "Attached are the meeting notes and action items from our QBR. Please review and confirm.",
  "The implementation is on track. Here's the updated timeline with key milestones.",
  "We've corrected the billing discrepancy. A credit has been applied to your next invoice.",
  "Great session today! Attached are the training materials and recorded session link.",
  "Exciting updates coming in Q3. Here's an early preview of features relevant to your use case.",
  "Attached is our SOC 2 report and data processing agreement for your compliance review.",
  "Your custom report is ready. Attached in CSV and PDF formats as requested.",
  "Here are your API credentials and integration guide. Our team is available for any questions.",
  "Attached is the renewal proposal with the multi-year pricing options we discussed.",
  "Great strategy session today. Here are the key takeaways and next steps.",
  "Your support ticket has been resolved. Please confirm everything is working as expected.",
  "Here's your onboarding checklist with 30/60/90 day milestones. Let's schedule a kickoff call.",
  "Thank you for your interest in a partnership. Attached is our partner program overview.",
];

const TASK_SUBJECTS = [
  "Send renewal proposal to {company}",
  "Follow up on expansion budget approval",
  "Schedule contract review call",
  "Prepare custom analytics demo",
  "Send SLA documentation",
  "Create onboarding plan for {company}",
  "Review and send pricing proposal",
  "Set up training session for {company}",
  "Prepare QBR presentation",
  "Follow up on legal contract review",
  "Send integration documentation",
  "Schedule executive briefing",
  "Compile ROI case studies",
  "Update CRM with meeting notes",
  "Prepare data migration checklist",
  "Review open support tickets for {company}",
  "Send product roadmap preview",
  "Coordinate internal handoff to CS team",
  "Draft custom implementation plan",
  "Schedule post-go-live check-in",
];

const MEETING_TITLES = [
  "{company} — Quarterly Business Review",
  "{company} — Compliance Review",
  "{company} — Contract Walkthrough",
  "{company} — Analytics Demo",
  "{company} — CTO Introduction",
  "{company} — Campus License Scope",
  "{company} — Onboarding Kickoff",
  "{company} — Technical Deep-Dive",
  "{company} — Renewal Discussion",
  "{company} — Executive Briefing",
  "{company} — Product Roadmap Review",
  "{company} — Implementation Planning",
  "{company} — Training Session",
  "{company} — Partnership Discussion",
  "{company} — Security Review",
  "{company} — Migration Planning",
  "{company} — Strategy Session",
  "{company} — Support Escalation Review",
  "{company} — Feature Prioritization",
  "{company} — Go-Live Readiness",
];

const TICKET_SUBJECTS = [
  "SSO login failing intermittently",
  "Bulk export timing out for large datasets",
  "Billing discrepancy on invoice",
  "Need custom report for compliance audit",
  "API rate limit hitting during peak hours",
  "Dashboard widgets not loading on Safari",
  "Account migration from legacy system",
  "SSL certificate renewal for custom domain",
  "Feature request: dark mode for portal",
  "Cannot add more than expected team members",
  "Webhook delivery failures to endpoint",
  "Data sync delay between modules",
  "Search functionality returning stale results",
  "Mobile app crashing on report generation",
  "Permission error on shared dashboards",
  "PDF export missing chart visualizations",
  "Two-factor authentication not sending codes",
  "Calendar integration sync issues",
  "Custom field validation not working",
  "Email notification delays",
];

const TICKET_BODIES = [
  "Users report being redirected to login page after successful SAML authentication. Happens ~20% of the time.",
  "Export jobs with >50k rows fail after 5 minutes with a 504 gateway timeout.",
  "Client was charged for extra seats but should have been billed per the contract amendment.",
  "Requires a SOC 2 compliance report showing all data access logs for the past 12 months.",
  "Integration hitting 429 errors between 9-11 AM EST. Current limit is 100 req/s, requesting increase.",
  "Several dashboard charts show blank white boxes in Safari 17.x. Works fine in Chrome and Firefox.",
  "Waiting for CSV export from old system so we can map and import historical data.",
  "Custom domain cert expires in 14 days. Need client to update DNS CNAME record.",
  "Multiple users have requested dark mode support for the customer portal.",
  "Getting an error when trying to invite team members. Error says 'seat limit reached' but plan allows more.",
  "Webhook events returning 502. Multiple events queued and undelivered.",
  "Changes in the CRM module take 15-20 minutes to appear in the reporting module. Expected near real-time.",
  "Search index appears out of date. Recently updated records not appearing in search results.",
  "App crashes when generating reports with >100 data points on iOS 17.",
  "Users with editor role cannot access shared dashboards. Permission check seems too restrictive.",
  "PDF exports are missing all chart images. Only table data is included in the output.",
  "SMS and authenticator app codes not being delivered. Affecting multiple users.",
  "Google Calendar events not syncing. Last successful sync was 3 days ago.",
  "Required field validation fires on optional fields. Blocking form submissions.",
  "Email notifications arriving 4-6 hours late. Affecting time-sensitive workflows.",
];

const DEAL_NAME_TEMPLATES = [
  "{company} — Enterprise Annual",
  "{company} — Platform Expansion",
  "{company} — Production Suite",
  "{company} — Compliance Module",
  "{company} — API Gateway",
  "{company} — Analytics Suite",
  "{company} — Monitoring Platform",
  "{company} — POS Integration",
  "{company} — Fleet Tracking",
  "{company} — Campus License",
  "{company} — Cloud Migration",
  "{company} — Data Platform",
  "{company} — Security Suite",
  "{company} — Automation Package",
  "{company} — Custom Integration",
];

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function pick<T>(arr: T[], seed: number): T {
  return arr[((seed % arr.length) + arr.length) % arr.length];
}

function fillTemplate(template: string, vars: { company?: string; contact?: string }): string {
  let result = template;
  if (vars.company) result = result.replace(/\{company\}/g, vars.company);
  if (vars.contact) result = result.replace(/\{contact\}/g, vars.contact);
  return result;
}

async function api<T = Record<string, unknown>>(
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const url = `${BASE_URL}${path}`;
  const res = await fetch(url, {
    method,
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${method} ${path} → ${res.status}: ${text}`);
  }
  return res.json() as Promise<T>;
}

interface Created {
  id: string;
  properties: Record<string, string>;
}

async function createObject(
  objectType: string,
  properties: Record<string, string>,
): Promise<Created> {
  return api<Created>("POST", `/crm/v3/objects/${objectType}`, { properties });
}

async function associate(
  fromType: string,
  fromId: string,
  toType: string,
  toId: string,
  typeId: number,
): Promise<void> {
  await api(
    "PUT",
    `/crm/v4/objects/${fromType}/${fromId}/associations/${toType}/${toId}`,
    [{ associationCategory: "HUBSPOT_DEFINED", associationTypeId: typeId }],
  );
}

async function associateBoth(
  typeA: string,
  idA: string,
  typeB: string,
  idB: string,
  aToB: number,
  bToA: number,
): Promise<void> {
  await associate(typeA, idA, typeB, idB, aToB);
  await associate(typeB, idB, typeA, idA, bToA);
}

async function associateDefault(
  fromType: string,
  fromId: string,
  toType: string,
  toId: string,
): Promise<void> {
  await api(
    "PUT",
    `/crm/v4/objects/${fromType}/${fromId}/associations/default/${toType}/${toId}`,
  );
}

function progress(msg: string) {
  process.stdout.write(msg);
}

function done(count?: number) {
  console.log(count !== undefined ? ` done (${count})` : " done");
}

// ---------------------------------------------------------------------------
// Pipeline stage lookup
// ---------------------------------------------------------------------------

interface PipelineStage {
  id: string;
  label: string;
}

interface Pipeline {
  id: string;
  label: string;
  stages: PipelineStage[];
}

interface PipelineResponse {
  results: Pipeline[];
}

async function fetchStageMap(objectType: string): Promise<{ pipelineId: string; stages: Map<string, string> }> {
  const data = await api<PipelineResponse>("GET", `/crm/v3/pipelines/${objectType}`);
  const pipeline = data.results[0];
  if (!pipeline) throw new Error(`No pipeline found for ${objectType}`);
  const stages = new Map<string, string>();
  for (const s of pipeline.stages) {
    stages.set(s.label, s.id);
  }
  return { pipelineId: pipeline.id, stages };
}

function requireStage(stages: Map<string, string>, label: string): string {
  const id = stages.get(label);
  if (!id) throw new Error(`Stage "${label}" not found. Available: ${[...stages.keys()].join(", ")}`);
  return id;
}

// ---------------------------------------------------------------------------
// Association type IDs
// ---------------------------------------------------------------------------

const ASSOC = {
  contactToCompany: 1,
  companyToContact: 2,
  contactToDeal: 3,
  dealToContact: 4,
  companyToDeal: 5,
  dealToCompany: 6,
  contactToTicket: 15,
  ticketToContact: 16,
  companyToTicket: 25,
  ticketToCompany: 26,
} as const;

// ---------------------------------------------------------------------------
// Lifecycle stage distribution
// ---------------------------------------------------------------------------

const LIFECYCLE_STAGES: { stage: string; weight: number }[] = [
  { stage: "subscriber", weight: 5 },
  { stage: "lead", weight: 20 },
  { stage: "marketingqualifiedlead", weight: 10 },
  { stage: "salesqualifiedlead", weight: 15 },
  { stage: "opportunity", weight: 15 },
  { stage: "customer", weight: 25 },
  { stage: "evangelist", weight: 10 },
];

function lifecycleForIndex(idx: number): string {
  const total = LIFECYCLE_STAGES.reduce((sum, s) => sum + s.weight, 0);
  const pos = idx % total;
  let cumulative = 0;
  for (const s of LIFECYCLE_STAGES) {
    cumulative += s.weight;
    if (pos < cumulative) return s.stage;
  }
  return "lead";
}

// ---------------------------------------------------------------------------
// Generation
// ---------------------------------------------------------------------------

interface CompanyGen {
  name: string;
  domain: string;
  industry: string;
  lifecyclestage: string;
}

function generateCompanies(count: number): CompanyGen[] {
  const result: CompanyGen[] = [];
  for (let i = 0; i < count; i++) {
    const prefix = pick(COMPANY_PREFIXES, i);
    const suffix = pick(COMPANY_SUFFIXES, i * 7 + 3);
    const name = `${prefix} ${suffix}`;
    const slug = name.toLowerCase().replace(/[^a-z0-9]+/g, "");
    const tld = pick(DOMAIN_TLDS, i * 11);
    result.push({
      name,
      domain: `${slug}${tld}`,
      industry: pick(INDUSTRIES, i),
      lifecyclestage: lifecycleForIndex(i),
    });
  }
  return result;
}

interface ContactGen {
  firstname: string;
  lastname: string;
  email: string;
  phone: string;
  company: string;
  lifecyclestage: string;
  companyIdx: number;
}

function generateContacts(companies: CompanyGen[]): ContactGen[] {
  const result: ContactGen[] = [];
  let globalIdx = 0;
  for (let ci = 0; ci < companies.length; ci++) {
    const co = companies[ci];
    const contactCount = 3 + (ci % 3); // 3, 4, or 5
    for (let j = 0; j < contactCount; j++) {
      const firstname = pick(FIRST_NAMES, globalIdx);
      const lastname = pick(LAST_NAMES, globalIdx * 3 + 7);
      const areaCode = pick(PHONE_AREA_CODES, globalIdx * 2);
      const phoneNum = String(100 + (globalIdx % 900)).padStart(4, "0");
      // ~30% of contacts get one stage ahead
      let stage = co.lifecyclestage;
      if (globalIdx % 10 < 3) {
        const stageNames = LIFECYCLE_STAGES.map((s) => s.stage);
        const idx = stageNames.indexOf(stage);
        if (idx < stageNames.length - 1) stage = stageNames[idx + 1];
      }
      result.push({
        firstname,
        lastname,
        email: `${firstname.toLowerCase()}.${lastname.toLowerCase().replace(/'/g, "")}@${co.domain}`,
        phone: `+1-${areaCode}-555-${phoneNum}`,
        company: co.name,
        lifecyclestage: stage,
        companyIdx: ci,
      });
      globalIdx++;
    }
  }
  return result;
}

// Deal stages with target distribution
const DEAL_STAGE_DIST: { label: string; target: number; amountMin: number; amountMax: number; closeFuture: boolean }[] = [
  { label: "Appointment Scheduled", target: 10, amountMin: 15000, amountMax: 120000, closeFuture: true },
  { label: "Qualified To Buy", target: 10, amountMin: 25000, amountMax: 200000, closeFuture: true },
  { label: "Presentation Scheduled", target: 8, amountMin: 30000, amountMax: 250000, closeFuture: true },
  { label: "Decision Maker Bought-In", target: 8, amountMin: 40000, amountMax: 300000, closeFuture: true },
  { label: "Contract Sent", target: 6, amountMin: 50000, amountMax: 350000, closeFuture: true },
  { label: "Closed Won", target: 12, amountMin: 30000, amountMax: 500000, closeFuture: false },
  { label: "Closed Lost", target: 6, amountMin: 10000, amountMax: 150000, closeFuture: false },
];

interface DealGen {
  dealname: string;
  stage: string;
  amount: string;
  closedate: string;
  companyIdx: number;
  contactIdx: number; // index into the contacts array (first contact of company)
}

function generateDeals(companies: CompanyGen[], contacts: ContactGen[]): DealGen[] {
  const result: DealGen[] = [];

  // Build company→first contact index
  const companyFirstContact: number[] = [];
  for (let ci = 0; ci < companies.length; ci++) {
    companyFirstContact.push(contacts.findIndex((c) => c.companyIdx === ci));
  }

  // Assign deals from stages
  let stageSlotIdx = 0;
  const stageSlots: string[] = [];
  for (const sd of DEAL_STAGE_DIST) {
    for (let i = 0; i < sd.target; i++) stageSlots.push(sd.label);
  }

  // Companies at opportunity+ always get deals; some earlier-stage companies too
  const eligibleCompanies: number[] = [];
  const opportunityStages = ["opportunity", "customer", "evangelist"];
  for (let ci = 0; ci < companies.length; ci++) {
    if (opportunityStages.includes(companies[ci].lifecyclestage)) {
      eligibleCompanies.push(ci);
    } else if (ci % 10 === 0) {
      // ~10% of non-opportunity companies
      eligibleCompanies.push(ci);
    }
  }

  for (let di = 0; di < stageSlots.length && di < 60; di++) {
    const stage = stageSlots[di];
    const sd = DEAL_STAGE_DIST.find((s) => s.label === stage)!;
    const ci = eligibleCompanies[di % eligibleCompanies.length];
    const co = companies[ci];
    const contactIdx = companyFirstContact[ci];
    const amount = sd.amountMin + ((di * 7919) % (sd.amountMax - sd.amountMin));

    let closedate: string;
    if (sd.closeFuture) {
      const months = 1 + (di % 6);
      const d = new Date(2026, 1 + months, 1 + (di % 28));
      closedate = d.toISOString().slice(0, 10);
    } else {
      const monthsAgo = 1 + (di % 6);
      const d = new Date(2026, 1 - monthsAgo, 1 + (di % 28));
      closedate = d.toISOString().slice(0, 10);
    }

    result.push({
      dealname: fillTemplate(pick(DEAL_NAME_TEMPLATES, di + stageSlotIdx), { company: co.name }),
      stage,
      amount: String(Math.round(amount / 1000) * 1000),
      closedate,
      companyIdx: ci,
      contactIdx,
    });
    stageSlotIdx++;
  }
  return result;
}

// Ticket stages with distribution
const TICKET_STAGE_DIST: { label: string; target: number }[] = [
  { label: "New", target: 12 },
  { label: "Waiting on contact", target: 8 },
  { label: "Waiting on us", target: 10 },
  { label: "Closed", target: 10 },
];

interface TicketGen {
  subject: string;
  content: string;
  stage: string;
  priority: string;
  companyIdx: number;
  contactIdx: number;
}

function generateTickets(companies: CompanyGen[], contacts: ContactGen[]): TicketGen[] {
  const result: TicketGen[] = [];
  const priorities = ["HIGH", "MEDIUM", "MEDIUM", "MEDIUM", "MEDIUM", "LOW", "LOW", "HIGH", "HIGH", "MEDIUM"];

  // Customer/evangelist companies get tickets
  const eligibleCompanies: number[] = [];
  for (let ci = 0; ci < companies.length; ci++) {
    if (companies[ci].lifecyclestage === "customer" || companies[ci].lifecyclestage === "evangelist") {
      eligibleCompanies.push(ci);
    }
  }

  const stageSlots: string[] = [];
  for (const sd of TICKET_STAGE_DIST) {
    for (let i = 0; i < sd.target; i++) stageSlots.push(sd.label);
  }

  for (let ti = 0; ti < stageSlots.length && ti < 40; ti++) {
    const ci = eligibleCompanies[ti % eligibleCompanies.length];
    const contactIdx = contacts.findIndex((c) => c.companyIdx === ci);
    const contactOffset = ti % (3 + (ci % 3)); // vary which contact

    result.push({
      subject: pick(TICKET_SUBJECTS, ti),
      content: pick(TICKET_BODIES, ti),
      stage: stageSlots[ti],
      priority: pick(priorities, ti),
      companyIdx: ci,
      contactIdx: contactIdx + contactOffset,
    });
  }
  return result;
}

// ---------------------------------------------------------------------------
// Engagement property schemas
// ---------------------------------------------------------------------------

interface PropertySchema {
  objectType: string;
  name: string;
  label: string;
  type: string;
  fieldType: string;
  groupName: string;
}

const ENGAGEMENT_PROPERTIES: PropertySchema[] = [
  { objectType: "notes", name: "hs_note_body", label: "Note Body", type: "string", fieldType: "textarea", groupName: "engagement_info" },
  { objectType: "calls", name: "hs_call_body", label: "Call Notes", type: "string", fieldType: "textarea", groupName: "engagement_info" },
  { objectType: "calls", name: "hs_call_direction", label: "Call Direction", type: "enumeration", fieldType: "select", groupName: "engagement_info" },
  { objectType: "calls", name: "hs_call_duration", label: "Call Duration", type: "number", fieldType: "number", groupName: "engagement_info" },
  { objectType: "emails", name: "hs_email_subject", label: "Email Subject", type: "string", fieldType: "text", groupName: "engagement_info" },
  { objectType: "emails", name: "hs_email_text", label: "Email Body", type: "string", fieldType: "textarea", groupName: "engagement_info" },
  { objectType: "tasks", name: "hs_task_subject", label: "Task Title", type: "string", fieldType: "text", groupName: "engagement_info" },
  { objectType: "tasks", name: "hs_task_body", label: "Task Notes", type: "string", fieldType: "textarea", groupName: "engagement_info" },
  { objectType: "tasks", name: "hs_task_status", label: "Task Status", type: "enumeration", fieldType: "select", groupName: "engagement_info" },
  { objectType: "meetings", name: "hs_meeting_title", label: "Meeting Name", type: "string", fieldType: "text", groupName: "engagement_info" },
  { objectType: "meetings", name: "hs_meeting_start_time", label: "Start Time", type: "datetime", fieldType: "date", groupName: "engagement_info" },
  { objectType: "meetings", name: "hs_meeting_end_time", label: "End Time", type: "datetime", fieldType: "date", groupName: "engagement_info" },
];

async function seedEngagementProperties(): Promise<void> {
  for (const prop of ENGAGEMENT_PROPERTIES) {
    try {
      await api("POST", `/crm/v3/properties/${prop.objectType}`, {
        name: prop.name,
        label: prop.label,
        type: prop.type,
        fieldType: prop.fieldType,
        groupName: prop.groupName,
      });
    } catch {
      // Property may already exist from Go seed — ignore 409 conflicts
    }
  }
}

// ---------------------------------------------------------------------------
// Engagement generation
// ---------------------------------------------------------------------------

interface EngagementGen {
  type: "notes" | "calls" | "emails" | "tasks" | "meetings";
  properties: Record<string, string>;
  contactIdx: number;
  companyIdx: number;
  dealIdx: number | null;
}

function generateEngagements(
  contacts: ContactGen[],
  companies: CompanyGen[],
  deals: DealGen[],
): EngagementGen[] {
  const result: EngagementGen[] = [];

  // Build set of deal-linked contact indices for "hero" contacts
  const dealContactIndices = new Set<number>();
  const contactToDeal = new Map<number, number>();
  for (let di = 0; di < deals.length; di++) {
    dealContactIndices.add(deals[di].contactIdx);
    contactToDeal.set(deals[di].contactIdx, di);
  }

  let engIdx = 0;
  for (let ci = 0; ci < contacts.length; ci++) {
    const contact = contacts[ci];
    const company = companies[contact.companyIdx];
    const isHero = dealContactIndices.has(ci);
    const engCount = isHero ? 3 + (ci % 3) : 1 + (ci % 3);
    const dealIdx = contactToDeal.get(ci) ?? null;
    const contactName = `${contact.firstname} ${contact.lastname}`;
    const vars = { company: company.name, contact: contactName };

    for (let j = 0; j < engCount; j++) {
      const typeRoll = engIdx % 5;
      engIdx++;

      if (typeRoll === 0) {
        // Note
        result.push({
          type: "notes",
          properties: {
            hs_note_body: fillTemplate(pick(NOTE_TEMPLATES, engIdx), vars),
          },
          contactIdx: ci,
          companyIdx: contact.companyIdx,
          dealIdx,
        });
      } else if (typeRoll === 1) {
        // Call
        const direction = engIdx % 2 === 0 ? "INBOUND" : "OUTBOUND";
        const duration = String(120 + ((engIdx * 137) % 2580));
        result.push({
          type: "calls",
          properties: {
            hs_call_body: fillTemplate(pick(CALL_SUMMARIES, engIdx), vars),
            hs_call_direction: direction,
            hs_call_duration: duration,
          },
          contactIdx: ci,
          companyIdx: contact.companyIdx,
          dealIdx,
        });
      } else if (typeRoll === 2) {
        // Email
        result.push({
          type: "emails",
          properties: {
            hs_email_subject: fillTemplate(pick(EMAIL_SUBJECTS, engIdx), vars),
            hs_email_text: fillTemplate(pick(EMAIL_BODIES, engIdx), vars),
          },
          contactIdx: ci,
          companyIdx: contact.companyIdx,
          dealIdx,
        });
      } else if (typeRoll === 3) {
        // Task
        const statusRoll = engIdx % 10;
        let status: string;
        if (statusRoll < 4) status = "NOT_STARTED";
        else if (statusRoll < 7) status = "IN_PROGRESS";
        else status = "COMPLETED";
        result.push({
          type: "tasks",
          properties: {
            hs_task_subject: fillTemplate(pick(TASK_SUBJECTS, engIdx), vars),
            hs_task_body: `Follow up with ${contactName} at ${company.name}.`,
            hs_task_status: status,
          },
          contactIdx: ci,
          companyIdx: contact.companyIdx,
          dealIdx,
        });
      } else {
        // Meeting
        const daysOut = 1 + (engIdx % 60);
        const hour = 9 + (engIdx % 8);
        const durationMin = 30 + (engIdx % 4) * 15; // 30, 45, 60, or 75 min
        const start = new Date(2026, 1, 9 + daysOut, hour, 0, 0);
        const end = new Date(start.getTime() + durationMin * 60 * 1000);
        result.push({
          type: "meetings",
          properties: {
            hs_meeting_title: fillTemplate(pick(MEETING_TITLES, engIdx), vars),
            hs_meeting_start_time: start.toISOString(),
            hs_meeting_end_time: end.toISOString(),
          },
          contactIdx: ci,
          companyIdx: contact.companyIdx,
          dealIdx,
        });
      }
    }
  }
  return result;
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

async function main() {
  console.log(`Seeding Notspot at ${BASE_URL}\n`);

  // 1. Reset
  progress("Resetting server...");
  await api("POST", "/_notspot/reset");
  done();

  // 2. Seed engagement property schemas (critical bug fix)
  progress("Registering engagement property schemas...");
  await seedEngagementProperties();
  done(ENGAGEMENT_PROPERTIES.length);

  // 3. Generate data
  const companiesData = generateCompanies(100);
  const contactsData = generateContacts(companiesData);
  // Deals and tickets generated after pipeline fetch

  // 4. Create companies
  progress("Creating companies");
  const companyIds: string[] = [];
  for (let i = 0; i < companiesData.length; i++) {
    const c = companiesData[i];
    const res = await createObject("companies", {
      name: c.name,
      domain: c.domain,
      industry: c.industry,
      lifecyclestage: c.lifecyclestage,
    });
    companyIds.push(res.id);
    if ((i + 1) % 10 === 0) process.stdout.write(".");
  }
  done(companyIds.length);

  // 5. Create contacts + associate to companies
  progress("Creating contacts");
  const contactIds: string[] = [];
  for (let i = 0; i < contactsData.length; i++) {
    const c = contactsData[i];
    const res = await createObject("contacts", {
      email: c.email,
      firstname: c.firstname,
      lastname: c.lastname,
      phone: c.phone,
      company: c.company,
      lifecyclestage: c.lifecyclestage,
    });
    contactIds.push(res.id);
    if ((i + 1) % 10 === 0) process.stdout.write(".");
  }
  done(contactIds.length);

  progress("Associating contacts ↔ companies");
  for (let i = 0; i < contactsData.length; i++) {
    const contactId = contactIds[i];
    const companyId = companyIds[contactsData[i].companyIdx];
    await associateBoth(
      "contacts", contactId,
      "companies", companyId,
      ASSOC.contactToCompany,
      ASSOC.companyToContact,
    );
    if ((i + 1) % 10 === 0) process.stdout.write(".");
  }
  done(contactsData.length);

  // 6. Fetch pipeline stage IDs
  progress("Fetching pipeline stages...");
  const dealPipeline = await fetchStageMap("deals");
  const ticketPipeline = await fetchStageMap("tickets");
  done();

  // 7. Generate deals and tickets (need pipeline data)
  const dealsData = generateDeals(companiesData, contactsData);
  const ticketsData = generateTickets(companiesData, contactsData);

  // 8. Create deals + associations
  progress("Creating deals");
  const dealIds: string[] = [];
  for (let i = 0; i < dealsData.length; i++) {
    const d = dealsData[i];
    const res = await createObject("deals", {
      dealname: d.dealname,
      dealstage: requireStage(dealPipeline.stages, d.stage),
      pipeline: dealPipeline.pipelineId,
      amount: d.amount,
      closedate: d.closedate,
    });
    dealIds.push(res.id);
    if ((i + 1) % 10 === 0) process.stdout.write(".");
  }
  done(dealIds.length);

  progress("Associating deals ↔ companies and contacts");
  for (let i = 0; i < dealsData.length; i++) {
    const dealId = dealIds[i];
    const companyId = companyIds[dealsData[i].companyIdx];
    const contactId = contactIds[dealsData[i].contactIdx];
    await associateBoth("deals", dealId, "companies", companyId, ASSOC.dealToCompany, ASSOC.companyToDeal);
    await associateBoth("deals", dealId, "contacts", contactId, ASSOC.dealToContact, ASSOC.contactToDeal);
  }
  done(dealsData.length);

  // 9. Create tickets + associations
  progress("Creating tickets");
  const ticketIds: string[] = [];
  for (let i = 0; i < ticketsData.length; i++) {
    const t = ticketsData[i];
    const res = await createObject("tickets", {
      subject: t.subject,
      content: t.content,
      hs_pipeline: ticketPipeline.pipelineId,
      hs_pipeline_stage: requireStage(ticketPipeline.stages, t.stage),
      hs_ticket_priority: t.priority,
    });
    ticketIds.push(res.id);
    if ((i + 1) % 10 === 0) process.stdout.write(".");
  }
  done(ticketIds.length);

  progress("Associating tickets ↔ companies and contacts");
  for (let i = 0; i < ticketsData.length; i++) {
    const ticketId = ticketIds[i];
    const companyId = companyIds[ticketsData[i].companyIdx];
    const contactId = contactIds[ticketsData[i].contactIdx];
    await associateBoth("tickets", ticketId, "companies", companyId, ASSOC.ticketToCompany, ASSOC.companyToTicket);
    await associateBoth("tickets", ticketId, "contacts", contactId, ASSOC.ticketToContact, ASSOC.contactToTicket);
  }
  done(ticketsData.length);

  // 10. Create engagements + associations
  const engagements = generateEngagements(contactsData, companiesData, dealsData);

  // Count by type
  const engCounts: Record<string, number> = {};
  for (const e of engagements) {
    engCounts[e.type] = (engCounts[e.type] || 0) + 1;
  }

  progress("Creating engagements");
  for (let i = 0; i < engagements.length; i++) {
    const e = engagements[i];
    const res = await createObject(e.type, e.properties);

    // Associate to contact
    await associateDefault(e.type, res.id, "contacts", contactIds[e.contactIdx]);
    // Associate to company
    await associateDefault(e.type, res.id, "companies", companyIds[e.companyIdx]);
    // Associate to deal if applicable
    if (e.dealIdx !== null) {
      await associateDefault(e.type, res.id, "deals", dealIds[e.dealIdx]);
    }

    if ((i + 1) % 10 === 0) process.stdout.write(".");
  }
  done(engagements.length);

  // Summary
  console.log("\nSeed complete!");
  console.log(`  Companies:  ${companyIds.length}`);
  console.log(`  Contacts:   ${contactIds.length}`);
  console.log(`  Deals:      ${dealIds.length}`);
  console.log(`  Tickets:    ${ticketIds.length}`);
  console.log(`  Notes:      ${engCounts["notes"] || 0}`);
  console.log(`  Calls:      ${engCounts["calls"] || 0}`);
  console.log(`  Emails:     ${engCounts["emails"] || 0}`);
  console.log(`  Tasks:      ${engCounts["tasks"] || 0}`);
  console.log(`  Meetings:   ${engCounts["meetings"] || 0}`);
  console.log(`  Total:      ${companyIds.length + contactIds.length + dealIds.length + ticketIds.length + engagements.length}`);
  console.log(`\nVisit ${BASE_URL}/_ui/ to browse the data.`);
}

main().catch((err) => {
  console.error("\nError:", err instanceof Error ? err.message : err);
  process.exit(1);
});
