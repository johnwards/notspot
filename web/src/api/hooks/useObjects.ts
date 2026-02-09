import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '../client';
import type { CrmObject, ObjectPage, CreateInput, SearchRequest, SearchResult } from '../types';

export const objectKeys = {
  all: ['objects'] as const,
  lists: (type: string) => [...objectKeys.all, type, 'list'] as const,
  list: (type: string, opts?: { limit?: number; after?: string; properties?: string[] }) =>
    [...objectKeys.lists(type), opts] as const,
  details: (type: string) => [...objectKeys.all, type, 'detail'] as const,
  detail: (type: string, id: string) => [...objectKeys.details(type), id] as const,
};

export function useObjects(
  objectType: string,
  opts?: { limit?: number; after?: string; properties?: string[]; enabled?: boolean },
) {
  const params = new URLSearchParams();
  if (opts?.limit) params.set('limit', String(opts.limit));
  if (opts?.after) params.set('after', opts.after);
  if (opts?.properties?.length) params.set('properties', opts.properties.join(','));
  const qs = params.toString();

  // Exclude 'enabled' from the query key
  const { enabled, ...keyOpts } = opts ?? {};

  return useQuery({
    queryKey: objectKeys.list(objectType, keyOpts),
    queryFn: () =>
      apiFetch<ObjectPage>(`/crm/v3/objects/${objectType}${qs ? `?${qs}` : ''}`),
    enabled: enabled ?? true,
  });
}

export function useObject(objectType: string, id: string, properties?: string[]) {
  const params = new URLSearchParams();
  if (properties?.length) params.set('properties', properties.join(','));
  const qs = params.toString();

  return useQuery({
    queryKey: properties?.length
      ? [...objectKeys.detail(objectType, id), properties]
      : objectKeys.detail(objectType, id),
    queryFn: () => apiFetch<CrmObject>(`/crm/v3/objects/${objectType}/${id}${qs ? `?${qs}` : ''}`),
    enabled: !!id,
  });
}

export function useCreateObject(objectType: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateInput) =>
      apiFetch<CrmObject>(`/crm/v3/objects/${objectType}`, {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: objectKeys.lists(objectType) });
    },
  });
}

export function useUpdateObject(objectType: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, properties }: { id: string; properties: Record<string, string> }) =>
      apiFetch<CrmObject>(`/crm/v3/objects/${objectType}/${id}`, {
        method: 'PATCH',
        body: JSON.stringify({ properties }),
      }),
    onSuccess: (_data, vars) => {
      void qc.invalidateQueries({ queryKey: objectKeys.detail(objectType, vars.id) });
      void qc.invalidateQueries({ queryKey: objectKeys.lists(objectType) });
    },
  });
}

export function useArchiveObject(objectType: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<void>(`/crm/v3/objects/${objectType}/${id}`, {
        method: 'DELETE',
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: objectKeys.lists(objectType) });
    },
  });
}

export function useSearchObjects(objectType: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (request: SearchRequest) =>
      apiFetch<SearchResult>(`/crm/v3/objects/${objectType}/search`, {
        method: 'POST',
        body: JSON.stringify(request),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: objectKeys.lists(objectType) });
    },
  });
}
