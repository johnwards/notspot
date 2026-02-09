import { useState, useEffect } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { apiFetch } from '@/api/client'
import type { SearchResult } from '@/api/types'
import { Loader2 } from 'lucide-react'

interface RecentRecord {
  id: string
  objectType: string
  name: string
}

const TYPES_TO_SEARCH = ['contacts', 'deals'] as const

const DISPLAY_PROPS: Record<string, string> = {
  contacts: 'email',
  deals: 'dealname',
}

export function RecentRecords() {
  const [records, setRecords] = useState<RecentRecord[]>([])
  const [loading, setLoading] = useState(true)
  const navigate = useNavigate()

  useEffect(() => {
    let cancelled = false

    async function fetchRecent() {
      try {
        const searches = TYPES_TO_SEARCH.map((objectType) =>
          apiFetch<SearchResult>(`/crm/v3/objects/${objectType}/search`, {
            method: 'POST',
            body: JSON.stringify({
              sorts: [{ propertyName: 'hs_lastmodifieddate', direction: 'DESCENDING' }],
              limit: 5,
            }),
          }).then((data) =>
            data.results.map((obj) => ({
              id: obj.id,
              objectType,
              name: obj.properties[DISPLAY_PROPS[objectType]] || `${objectType} #${obj.id}`,
            })),
          ).catch(() => [] as RecentRecord[]),
        )

        const groups = await Promise.all(searches)
        if (!cancelled) {
          setRecords(groups.flat())
          setLoading(false)
        }
      } catch {
        if (!cancelled) setLoading(false)
      }
    }

    void fetchRecent()
    return () => { cancelled = true }
  }, [])

  return (
    <Card data-testid="recent-records">
      <CardHeader>
        <CardTitle>Recent Records</CardTitle>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="flex items-center justify-center py-4">
            <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
          </div>
        ) : records.length === 0 ? (
          <p className="text-sm text-muted-foreground">No records yet.</p>
        ) : (
          <ul className="space-y-2">
            {records.map((r) => (
              <li key={`${r.objectType}-${r.id}`}>
                <button
                  className="flex w-full items-center justify-between rounded-md px-2 py-1.5 text-sm hover:bg-muted transition-colors text-left"
                  onClick={() => {
                    void navigate({
                      to: '/$objectType/$objectId',
                      params: { objectType: r.objectType, objectId: r.id },
                    })
                  }}
                >
                  <span className="truncate">{r.name}</span>
                  <span className="ml-2 shrink-0 text-xs text-muted-foreground capitalize">{r.objectType}</span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  )
}
