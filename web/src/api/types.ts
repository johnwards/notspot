// Object (from internal/domain/object.go)
export interface CrmObject {
  id: string;
  properties: Record<string, string>;
  createdAt: string;
  updatedAt: string;
  archived: boolean;
  archivedAt?: string;
}

export interface CreateInput {
  properties: Record<string, string>;
}

export interface UpdateInput {
  id: string;
  properties: Record<string, string>;
}

export interface ObjectPage {
  results: CrmObject[];
  paging?: { next?: { after: string } };
}

export interface BatchResult {
  status: string;
  results: CrmObject[];
  startedAt: string;
  completedAt: string;
  numErrors?: number;
}

// Property (from internal/domain/property.go)
export interface Property {
  name: string;
  label: string;
  type: string;
  fieldType: string;
  groupName: string;
  description: string;
  options: PropertyOption[];
  displayOrder: number;
  hasUniqueValue: boolean;
  hidden: boolean;
  formField: boolean;
  calculated: boolean;
  externalOptions: boolean;
  archived: boolean;
  hubspotDefined: boolean;
  createdAt?: string;
  updatedAt?: string;
  archivedAt?: string;
  modificationMetadata?: ModificationMetadata;
}

export interface PropertyOption {
  label: string;
  value: string;
  displayOrder: number;
  hidden: boolean;
}

export interface ModificationMetadata {
  archivable: boolean;
  readOnlyDefinition: boolean;
  readOnlyOptions: boolean;
  readOnlyValue: boolean;
}

export interface PropertyGroup {
  name: string;
  label: string;
  displayOrder: number;
  archived: boolean;
}

// Pipeline (from internal/domain/pipeline.go)
export interface Pipeline {
  id: string;
  label: string;
  displayOrder: number;
  stages: PipelineStage[];
  archived: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface PipelineStage {
  id: string;
  label: string;
  displayOrder: number;
  metadata: Record<string, string>;
  archived: boolean;
  createdAt: string;
  updatedAt: string;
}

// Association (from internal/domain/association.go)
export interface AssociationType {
  typeId: number;
  label: string;
  category: string;
}

export interface AssociationResult {
  toObjectId: string;
  associationTypes: AssociationType[];
}

export interface AssociationLabel {
  category: string;
  typeId: number;
  label: string;
}

// Schema (from internal/domain/schema.go)
export interface ObjectSchema {
  id: string;
  name: string;
  labels: SchemaLabels;
  primaryDisplayProperty: string;
  properties: Property[];
  associations: SchemaAssociation[];
  associatedObjects?: string[];
  fullyQualifiedName: string;
  archived: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface SchemaLabels {
  singular: string;
  plural: string;
}

export interface SchemaAssociation {
  id: string;
  fromObjectTypeId: string;
  toObjectTypeId: string;
  name?: string;
  labels?: SchemaLabels;
  createdAt?: string;
  updatedAt?: string;
}

// Search (from internal/domain/search.go)
export interface SearchRequest {
  query?: string;
  filterGroups?: FilterGroup[];
  sorts?: Sort[];
  properties?: string[];
  limit?: number;
  after?: string;
}

export interface FilterGroup {
  filters: Filter[];
}

export interface Filter {
  propertyName: string;
  operator: string;
  value?: string;
  highValue?: string;
  values?: string[];
}

export interface Sort {
  propertyName: string;
  direction: string;
}

export interface SearchResult {
  total: number;
  results: CrmObject[];
  paging?: { next: { after: string } };
}

// List (from internal/domain/list.go)
export interface HubSpotList {
  listId: string;
  name: string;
  objectTypeId: string;
  processingType: string;
  processingStatus: string;
  filterBranch?: unknown;
  listVersion: number;
  size: number;
  createdAt: string;
  updatedAt: string;
}

// Error format
export interface HubSpotError {
  status: string;
  message: string;
  correlationId: string;
  category: string;
  errors?: ErrorDetail[];
}

export interface ErrorDetail {
  message: string;
  code?: string;
  in?: string;
  context?: Record<string, string[]>;
  subCategory?: string;
}

// Owner
export interface Owner {
  id: string;
  email: string;
  firstName: string;
  lastName: string;
  userId?: number;
  createdAt: string;
  updatedAt: string;
  archived: boolean;
}
