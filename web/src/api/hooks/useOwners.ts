import { useQuery } from '@tanstack/react-query';
import { apiFetch } from '../client';
import type { Owner } from '../types';

export const ownerKeys = {
  all: ['owners'] as const,
  detail: (id: string) => [...ownerKeys.all, id] as const,
};

export function useOwners() {
  return useQuery({
    queryKey: ownerKeys.all,
    queryFn: () => apiFetch<{ results: Owner[] }>('/crm/v3/owners'),
  });
}

export function useOwner(id: string) {
  return useQuery({
    queryKey: ownerKeys.detail(id),
    queryFn: () => apiFetch<Owner>(`/crm/v3/owners/${id}`),
    enabled: !!id,
  });
}
