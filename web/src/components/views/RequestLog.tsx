import { useState, useEffect, useRef } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Activity, ChevronDown, ChevronRight, RefreshCw } from 'lucide-react'
import { apiFetch } from '@/api/client'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/shared/EmptyState'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

interface RequestEntry {
  method: string
  path: string
  status: number
  timestamp: string
  requestBody?: unknown
  responseBody?: unknown
}

function methodColor(method: string): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (method.toUpperCase()) {
    case 'GET':
      return 'secondary'
    case 'POST':
      return 'default'
    case 'DELETE':
      return 'destructive'
    default:
      return 'outline'
  }
}

function statusClassName(status: number): string {
  if (status >= 200 && status < 300) return 'text-green-600'
  if (status >= 400 && status < 500) return 'text-yellow-600'
  if (status >= 500) return 'text-red-600'
  return ''
}

export function RequestLog() {
  const [autoRefresh, setAutoRefresh] = useState(false)
  const [expandedRows, setExpandedRows] = useState<Set<number>>(new Set())
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['admin', 'requests'],
    queryFn: () => apiFetch<{ results: RequestEntry[] }>('/_notspot/requests'),
    retry: false,
  })

  useEffect(() => {
    if (autoRefresh) {
      intervalRef.current = setInterval(() => {
        void refetch()
      }, 2000)
    }
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
        intervalRef.current = null
      }
    }
  }, [autoRefresh, refetch])

  const toggleRow = (idx: number) => {
    setExpandedRows((prev) => {
      const next = new Set(prev)
      if (next.has(idx)) {
        next.delete(idx)
      } else {
        next.add(idx)
      }
      return next
    })
  }

  if (isError) {
    return (
      <EmptyState
        icon={Activity}
        title="Request logging not available"
        description="The /_notspot/requests endpoint is not available on this server."
      />
    )
  }

  const requests = data?.results ?? []

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Request Log</h2>
        <div className="flex items-center gap-2">
          <Button
            variant={autoRefresh ? 'default' : 'outline'}
            size="sm"
            onClick={() => setAutoRefresh((v) => !v)}
          >
            <RefreshCw className={cn('mr-1 h-4 w-4', autoRefresh && 'animate-spin')} />
            {autoRefresh ? 'Auto-refresh on' : 'Auto-refresh off'}
          </Button>
        </div>
      </div>

      {isLoading ? (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-8" />
              <TableHead>Method</TableHead>
              <TableHead>Path</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Time</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {Array.from({ length: 5 }).map((_, i) => (
              <TableRow key={i}>
                <TableCell><Skeleton className="h-4 w-4" /></TableCell>
                <TableCell><Skeleton className="h-4 w-12" /></TableCell>
                <TableCell><Skeleton className="h-4 w-48" /></TableCell>
                <TableCell><Skeleton className="h-4 w-10" /></TableCell>
                <TableCell><Skeleton className="h-4 w-24" /></TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      ) : requests.length === 0 ? (
        <EmptyState
          icon={Activity}
          title="No requests"
          description="No requests have been logged yet."
        />
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-8" />
              <TableHead>Method</TableHead>
              <TableHead>Path</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Time</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {requests.map((req, idx) => (
              <>
                <TableRow
                  key={idx}
                  className="cursor-pointer"
                  onClick={() => toggleRow(idx)}
                >
                  <TableCell>
                    {expandedRows.has(idx) ? (
                      <ChevronDown className="h-4 w-4" />
                    ) : (
                      <ChevronRight className="h-4 w-4" />
                    )}
                  </TableCell>
                  <TableCell>
                    <Badge variant={methodColor(req.method)}>
                      {req.method}
                    </Badge>
                  </TableCell>
                  <TableCell className="font-mono text-sm">{req.path}</TableCell>
                  <TableCell>
                    <span className={cn('font-mono font-medium', statusClassName(req.status))}>
                      {req.status}
                    </span>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {req.timestamp ? new Date(req.timestamp).toLocaleTimeString() : '-'}
                  </TableCell>
                </TableRow>
                {expandedRows.has(idx) && (
                  <TableRow key={`${idx}-detail`}>
                    <TableCell colSpan={5} className="bg-muted/50 p-4">
                      <div className="grid grid-cols-2 gap-4">
                        <div>
                          <p className="mb-1 text-xs font-medium text-muted-foreground">
                            Request Body
                          </p>
                          <pre className="max-h-48 overflow-auto rounded bg-muted p-2 text-xs">
                            {req.requestBody
                              ? JSON.stringify(req.requestBody, null, 2)
                              : '(empty)'}
                          </pre>
                        </div>
                        <div>
                          <p className="mb-1 text-xs font-medium text-muted-foreground">
                            Response Body
                          </p>
                          <pre className="max-h-48 overflow-auto rounded bg-muted p-2 text-xs">
                            {req.responseBody
                              ? JSON.stringify(req.responseBody, null, 2)
                              : '(empty)'}
                          </pre>
                        </div>
                      </div>
                    </TableCell>
                  </TableRow>
                )}
              </>
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  )
}
