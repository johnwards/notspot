import { useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '../client';
import { associationKeys } from './useAssociations';

export function useCreateAssociation(fromType: string, fromId: string, toType: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (toId: string) =>
      apiFetch<unknown>(`/crm/v4/objects/${fromType}/${fromId}/associations/default/${toType}/${toId}`, {
        method: 'PUT',
        body: JSON.stringify({}),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: associationKeys.all });
    },
  });
}

export function useRemoveAssociation(fromType: string, fromId: string, toType: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (toId: string) =>
      apiFetch<void>(`/crm/v4/objects/${fromType}/${fromId}/associations/${toType}/${toId}`, {
        method: 'DELETE',
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: associationKeys.all });
    },
  });
}
