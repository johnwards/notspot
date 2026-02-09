import { useQuery } from '@tanstack/react-query';
import { apiFetch } from '../client';
import type { AssociationResult, AssociationLabel } from '../types';

export const associationKeys = {
  all: ['associations'] as const,
  list: (fromType: string, objectId: string, toType: string) =>
    [...associationKeys.all, fromType, objectId, toType] as const,
  labels: (fromType: string, toType: string) =>
    [...associationKeys.all, 'labels', fromType, toType] as const,
};

export function useAssociations(fromType: string, objectId: string, toType: string) {
  return useQuery({
    queryKey: associationKeys.list(fromType, objectId, toType),
    queryFn: () =>
      apiFetch<{ results: AssociationResult[] }>(
        `/crm/v4/objects/${fromType}/${objectId}/associations/${toType}`,
      ),
    enabled: !!objectId,
  });
}

export function useAssociationLabels(fromType: string, toType: string) {
  return useQuery({
    queryKey: associationKeys.labels(fromType, toType),
    queryFn: () =>
      apiFetch<{ results: AssociationLabel[] }>(
        `/crm/v4/associations/${fromType}/${toType}/labels`,
      ),
  });
}
