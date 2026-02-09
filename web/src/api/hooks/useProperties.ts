import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '../client';
import type { Property, PropertyGroup } from '../types';

export const propertyKeys = {
  all: ['properties'] as const,
  lists: (type: string) => [...propertyKeys.all, type] as const,
  groups: (type: string) => [...propertyKeys.all, type, 'groups'] as const,
};

export function useProperties(objectType: string) {
  return useQuery({
    queryKey: propertyKeys.lists(objectType),
    queryFn: () =>
      apiFetch<{ results: Property[] }>(`/crm/v3/properties/${objectType}`),
  });
}

export function useCreateProperty(objectType: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: Partial<Property>) =>
      apiFetch<Property>(`/crm/v3/properties/${objectType}`, {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: propertyKeys.lists(objectType) });
    },
  });
}

export function useUpdateProperty(objectType: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ propertyName, ...input }: Partial<Property> & { propertyName: string }) =>
      apiFetch<Property>(`/crm/v3/properties/${objectType}/${propertyName}`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: propertyKeys.lists(objectType) });
    },
  });
}

export function useArchiveProperty(objectType: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (propertyName: string) =>
      apiFetch<void>(`/crm/v3/properties/${objectType}/${propertyName}`, {
        method: 'DELETE',
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: propertyKeys.lists(objectType) });
    },
  });
}

export function usePropertyGroups(objectType: string) {
  return useQuery({
    queryKey: propertyKeys.groups(objectType),
    queryFn: () =>
      apiFetch<{ results: PropertyGroup[] }>(`/crm/v3/properties/${objectType}/groups`),
  });
}
