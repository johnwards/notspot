import { useState } from 'react'
import { GitBranch } from 'lucide-react'
import { usePipelines } from '@/api/hooks/usePipelines'
import type { Pipeline } from '@/api/types'
import { EmptyState } from '@/components/shared/EmptyState'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Skeleton } from '@/components/ui/skeleton'

const PIPELINE_OBJECT_TYPES = ['deals', 'tickets'] as const

function PipelineCard({ pipeline }: { pipeline: Pipeline }) {
  const sortedStages = [...pipeline.stages].sort(
    (a, b) => a.displayOrder - b.displayOrder,
  )

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          {pipeline.label}
          <Badge variant="outline">ID: {pipeline.id}</Badge>
        </CardTitle>
        <CardDescription>
          {pipeline.stages.length} stage{pipeline.stages.length !== 1 ? 's' : ''} &middot;
          Display order: {pipeline.displayOrder}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[60px]">Order</TableHead>
              <TableHead>Label</TableHead>
              <TableHead>ID</TableHead>
              <TableHead>Metadata</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {sortedStages.map((stage) => (
              <TableRow key={stage.id}>
                <TableCell className="font-mono text-muted-foreground">
                  {stage.displayOrder}
                </TableCell>
                <TableCell className="font-medium">{stage.label}</TableCell>
                <TableCell className="text-muted-foreground">{stage.id}</TableCell>
                <TableCell>
                  {Object.keys(stage.metadata).length > 0 ? (
                    <div className="flex flex-wrap gap-1">
                      {Object.entries(stage.metadata).map(([key, value]) => (
                        <Badge key={key} variant="secondary" className="text-xs">
                          {key}: {value}
                        </Badge>
                      ))}
                    </div>
                  ) : (
                    <span className="text-muted-foreground">-</span>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  )
}

export function PipelineSettings() {
  const [objectType, setObjectType] = useState<string>('deals')
  const { data, isLoading } = usePipelines(objectType)

  const pipelines = data?.results ?? []

  return (
    <div className="space-y-6 p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Pipelines</h1>
          <p className="text-sm text-muted-foreground">
            View pipeline stages for your CRM objects
          </p>
        </div>
        <Select value={objectType} onValueChange={setObjectType}>
          <SelectTrigger className="w-[160px]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {PIPELINE_OBJECT_TYPES.map((t) => (
              <SelectItem key={t} value={t}>
                {t.charAt(0).toUpperCase() + t.slice(1)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {isLoading ? (
        <div className="space-y-4">
          {Array.from({ length: 2 }).map((_, i) => (
            <Card key={i}>
              <CardHeader>
                <Skeleton className="h-5 w-48" />
                <Skeleton className="h-4 w-32" />
              </CardHeader>
              <CardContent>
                <Skeleton className="h-32 w-full" />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : pipelines.length === 0 ? (
        <EmptyState
          icon={GitBranch}
          title="No pipelines"
          description={`No pipelines found for ${objectType}.`}
        />
      ) : (
        <div className="space-y-4">
          {pipelines.map((pipeline) => (
            <PipelineCard key={pipeline.id} pipeline={pipeline} />
          ))}
        </div>
      )}
    </div>
  )
}
