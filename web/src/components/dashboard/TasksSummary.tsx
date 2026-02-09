import { useState, useEffect } from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { apiFetch } from '@/api/client'
import type { SearchResult } from '@/api/types'
import { CheckSquare, Loader2 } from 'lucide-react'

export function TasksSummary() {
  const [openCount, setOpenCount] = useState(0)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false

    async function fetchTasks() {
      try {
        const data = await apiFetch<SearchResult>('/crm/v3/objects/tasks/search', {
          method: 'POST',
          body: JSON.stringify({
            filterGroups: [
              {
                filters: [
                  { propertyName: 'hs_task_status', operator: 'NEQ', value: 'COMPLETED' },
                ],
              },
            ],
            limit: 1,
          }),
        })

        if (!cancelled) {
          setOpenCount(data.total)
          setLoading(false)
        }
      } catch {
        if (!cancelled) setLoading(false)
      }
    }

    void fetchTasks()
    return () => { cancelled = true }
  }, [])

  return (
    <Card data-testid="tasks-summary">
      <CardHeader>
        <CardTitle>Open Tasks</CardTitle>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="flex items-center justify-center py-4">
            <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
          </div>
        ) : (
          <div className="flex items-center gap-3">
            <CheckSquare className="h-8 w-8 text-muted-foreground" />
            <div>
              <p className="text-3xl font-bold">{openCount}</p>
              <p className="text-sm text-muted-foreground">tasks remaining</p>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
