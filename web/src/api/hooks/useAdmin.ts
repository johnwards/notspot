import { useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '../client';

export function useResetData() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiFetch<void>('/_notspot/reset', { method: 'POST' }),
    onSuccess: () => {
      void qc.invalidateQueries();
    },
  });
}
