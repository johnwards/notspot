import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider, MutationCache } from '@tanstack/react-query'
import { toast } from 'sonner'
import { NetworkError } from '@/api/client'
import './index.css'
import App from './App'

const queryClient = new QueryClient({
  mutationCache: new MutationCache({
    onError: (error, _variables, _context, mutation) => {
      // Only show toast if the mutation call doesn't have its own onError handler
      if (!mutation.options.onError) {
        if (error instanceof NetworkError) {
          toast.error('Network error — unable to reach the server');
        } else {
          toast.error(`Error: ${error.message}`);
        }
      }
    },
  }),
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: (failureCount, error) => {
        if (error instanceof NetworkError) return failureCount < 2;
        return failureCount < 1;
      },
    },
  },
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </StrictMode>,
)
