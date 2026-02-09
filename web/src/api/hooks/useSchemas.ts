import { useQuery } from '@tanstack/react-query';
import { apiFetch } from '../client';
import type { ObjectSchema } from '../types';

export const schemaKeys = {
  all: ['schemas'] as const,
};

export function useSchemas() {
  return useQuery({
    queryKey: schemaKeys.all,
    queryFn: () => apiFetch<{ results: ObjectSchema[] }>('/crm/v3/schemas'),
  });
}
