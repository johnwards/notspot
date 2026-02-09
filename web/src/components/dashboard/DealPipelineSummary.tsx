import { useState, useEffect } from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { apiFetch } from '@/api/client'
import { usePipelines } from '@/api/hooks/usePipelines'
import type { SearchResult } from '@/api/types'
import { Loader2 } from 'lucide-react'

export function DealPipelineSummary() {
  const [stageCounts, setStageCounts] = useState<Record<string, number>>({})
  const [loading, setLoading] = useState(true)
  const { data: pipelinesData } = usePipelines('deals')

  useEffect(() => {
    let cancelled = false

    async function fetchDeals() {
      try {
        const data = await apiFetch<SearchResult>('/crm/v3/objects/deals/search', {
          method: 'POST',
          body: JSON.stringify({ properties: ['dealstage'], limit: 200 }),
        })

        if (!cancelled) {
          const counts: Record<string, number> = {}
          for (const deal of data.results) {
            const stage = deal.properties.dealstage || 'unknown'
            counts[stage] = (counts[stage] || 0) + 1
          }
          setStageCounts(counts)
          setLoading(false)
        }
      } catch {
        if (!cancelled) setLoading(false)
      }
    }

    void fetchDeals()
    return () => { cancelled = true }
  }, [])

  // Build a stage ID → label map from pipelines
  const stageLabels: Record<string, string> = {}
  if (pipelinesData?.results) {
    for (const pipeline of pipelinesData.results) {
      for (const stage of pipeline.stages) {
        stageLabels[stage.id] = stage.label
      }
    }
  }

  return (
    <Card data-testid="deal-pipeline-summary">
      <CardHeader>
        <CardTitle>Deal Pipeline</CardTitle>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="flex items-center justify-center py-4">
            <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
          </div>
        ) : Object.keys(stageCounts).length === 0 ? (
          <p className="text-sm text-muted-foreground">No deals yet.</p>
        ) : (
          <ul className="space-y-2">
            {Object.entries(stageCounts).map(([stageId, count]) => (
              <li key={stageId} className="flex items-center justify-between text-sm">
                <span>{stageLabels[stageId] || stageId}</span>
                <span className="font-medium">{count}</span>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  )
}
