import { useQuery } from '@tanstack/react-query';
import { apiFetch } from '../client';
import type { Pipeline } from '../types';

export const pipelineKeys = {
  all: ['pipelines'] as const,
  lists: (type: string) => [...pipelineKeys.all, type] as const,
};

export function usePipelines(objectType: string) {
  return useQuery({
    queryKey: pipelineKeys.lists(objectType),
    queryFn: () =>
      apiFetch<{ results: Pipeline[] }>(`/crm/v3/pipelines/${objectType}`),
  });
}
