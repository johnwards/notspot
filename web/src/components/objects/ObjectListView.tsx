import { useState, useCallback, useEffect, useMemo } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { useObjects, useSearchObjects } from '@/api/hooks/useObjects'
import { useProperties } from '@/api/hooks/useProperties'
import { useOwners } from '@/api/hooks/useOwners'
import { useQueryClient } from '@tanstack/react-query'
import { objectKeys } from '@/api/hooks/useObjects'
import { DataTable, type Column } from '@/components/shared/DataTable'
import { Pagination } from '@/components/shared/Pagination'
import { EmptyState } from '@/components/shared/EmptyState'
import { TableSkeleton } from '@/components/shared/LoadingSkeleton'
import { ObjectCreateDialog } from './ObjectCreateDialog'
import { EditColumnsDialog } from './EditColumnsDialog'
import { BulkActionsBar } from './BulkActionsBar'
import { FilterPanel } from '@/components/filters/FilterPanel'
import { SavedViewTabs } from '@/components/filters/SavedViewTabs'
import { getColumnPreferences, saveColumnPreferences, clearColumnPreferences } from '@/lib/column-preferences'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Plus, Inbox, Search, Filter, Columns } from 'lucide-react'
import type { CrmObject, Property, FilterGroup } from '@/api/types'
import { singularize } from '@/lib/utils'

interface ObjectListViewProps {
  objectType: string
}

const PAGE_SIZE = 20

export function ObjectListView({ objectType }: ObjectListViewProps) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [afterStack, setAfterStack] = useState<string[]>([])
  const [searchQuery, setSearchQuery] = useState('')
  const [debouncedQuery, setDebouncedQuery] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [activeFilters, setActiveFilters] = useState<FilterGroup[]>([])
  const [activeViewId, setActiveViewId] = useState<string | null>(null)
  const [showFilterPanel, setShowFilterPanel] = useState(false)
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [columnsDialogOpen, setColumnsDialogOpen] = useState(false)
  const [customColumns, setCustomColumns] = useState<string[] | null>(null)

  const currentAfter = afterStack.length > 0 ? afterStack[afterStack.length - 1] : undefined

  const { data: propertiesData, isLoading: propsLoading } = useProperties(objectType)
  const { data: ownersData } = useOwners()

  // All non-hidden/non-archived properties sorted by displayOrder
  const allVisibleProps = useMemo<Property[]>(() => {
    if (!propertiesData?.results) return []
    return propertiesData.results
      .filter((p) => !p.hidden && !p.archived)
      .sort((a, b) => a.displayOrder - b.displayOrder)
  }, [propertiesData])

  // Default columns: first 6, plus hubspot_owner_id if not already included
  const defaultColumns = useMemo(() => {
    const cols = allVisibleProps.slice(0, 6).map((p) => p.name)
    if (!cols.includes('hubspot_owner_id') && allVisibleProps.some((p) => p.name === 'hubspot_owner_id')) {
      cols.push('hubspot_owner_id')
    }
    return cols
  }, [allVisibleProps])

  // Load saved column preferences on mount and when objectType changes
  useEffect(() => {
    const saved = getColumnPreferences(objectType)
    setCustomColumns(saved)
  }, [objectType])

  const activeColumnNames = customColumns ?? defaultColumns

  const visibleProps = useMemo<Property[]>(() => {
    return activeColumnNames
      .map((name) => allVisibleProps.find((p) => p.name === name))
      .filter((p): p is Property => p !== undefined)
  }, [activeColumnNames, allVisibleProps])

  const allFilterableProps = useMemo<Property[]>(() => {
    if (!propertiesData?.results) return []
    return propertiesData.results.filter((p) => !p.hidden && !p.archived)
  }, [propertiesData])

  const propertyNames = useMemo(() => visibleProps.map((p) => p.name), [visibleProps])

  const propsReady = propertyNames.length > 0

  // Count active filters
  const filterCount = useMemo(() => {
    return activeFilters.reduce((sum, group) => sum + (group.filters?.length ?? 0), 0)
  }, [activeFilters])

  // Determine if we should use the search endpoint
  const hasActiveFilters = filterCount > 0
  const hasSearchQuery = !!debouncedQuery.trim()
  const useSearch = hasActiveFilters || hasSearchQuery

  const { data: objectsData, isLoading: objectsLoading } = useObjects(objectType, {
    limit: PAGE_SIZE,
    after: currentAfter,
    properties: propsReady ? propertyNames : undefined,
    enabled: propsReady && !useSearch,
  })
  const searchMutation = useSearchObjects(objectType)

  // Debounce search
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedQuery(searchQuery), 300)
    return () => clearTimeout(timer)
  }, [searchQuery])

  // Execute search when debounced query or filters change
  useEffect(() => {
    if (useSearch) {
      searchMutation.mutate({
        query: debouncedQuery.trim() || undefined,
        filterGroups: hasActiveFilters ? activeFilters : undefined,
        limit: PAGE_SIZE,
        properties: propertyNames,
      })
    }
    setSelectedIds(new Set())
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [debouncedQuery, activeFilters, useSearch])

  // Reset pagination when objectType changes
  useEffect(() => {
    setAfterStack([])
    setSearchQuery('')
    setDebouncedQuery('')
    setActiveFilters([])
    setActiveViewId(null)
    setShowFilterPanel(false)
    setSelectedIds(new Set())
  }, [objectType])

  const columns = useMemo<Column<Record<string, unknown>>[]>(() => {
    return visibleProps.map((p) => ({
      key: p.name,
      header: p.label,
      sortable: true,
      render: (row: Record<string, unknown>) => {
        const props = row.properties as Record<string, string> | undefined
        const val = props?.[p.name] ?? ''
        if (p.name === 'hubspot_owner_id' && val && ownersData?.results) {
          const owner = ownersData.results.find((o) => o.id === val)
          return owner ? `${owner.firstName} ${owner.lastName}` : val
        }
        return val
      },
    }))
  }, [visibleProps, ownersData])

  const handleRowClick = useCallback((row: Record<string, unknown>) => {
    void navigate({
      to: '/$objectType/$objectId',
      params: { objectType, objectId: row.id as string },
    })
  }, [navigate, objectType])

  const handleNext = () => {
    const nextAfter = displayData?.paging?.next?.after
    if (nextAfter) {
      setAfterStack((prev) => [...prev, nextAfter])
      setSelectedIds(new Set())
    }
  }

  const handlePrevious = () => {
    setAfterStack((prev) => prev.slice(0, -1))
    setSelectedIds(new Set())
  }

  const handleFilterChange = useCallback((groups: FilterGroup[]) => {
    setActiveFilters(groups)
    setAfterStack([])
    setSelectedIds(new Set())
    // If we clear all filters while in "All" view, that's fine
    // If we had a saved view active and change filters, deactivate the view
    setActiveViewId(null)
  }, [])

  const handleViewChange = useCallback((viewId: string | null, filterGroups: FilterGroup[]) => {
    setActiveViewId(viewId)
    setActiveFilters(filterGroups)
    setAfterStack([])
    if (filterGroups.length > 0) {
      setShowFilterPanel(true)
    }
  }, [])

  const handleSaveColumns = useCallback((columns: string[]) => {
    saveColumnPreferences(objectType, columns)
    setCustomColumns(columns)
  }, [objectType])

  const handleResetColumns = useCallback(() => {
    clearColumnPreferences(objectType)
    setCustomColumns(null)
  }, [objectType])

  // Use search results if searching/filtering, otherwise regular list
  const displayData = useSearch
    ? searchMutation.data
      ? { results: searchMutation.data.results, paging: searchMutation.data.paging ? { next: searchMutation.data.paging.next } : undefined }
      : undefined
    : objectsData

  const isLoading = propsLoading || (!useSearch && objectsLoading) || (useSearch && searchMutation.isPending)
  const objects = displayData?.results ?? []

  // Map CrmObjects to rows for the DataTable
  const rows: Record<string, unknown>[] = objects.map((obj: CrmObject) => ({
    id: obj.id,
    properties: obj.properties,
    ...obj.properties,
  }))

  if (isLoading && !displayData) {
    return <TableSkeleton rows={8} columns={columns.length || 4} />
  }

  return (
    <div className="space-y-4">
      {/* Saved View Tabs */}
      <SavedViewTabs
        objectType={objectType}
        activeViewId={activeViewId}
        onViewChange={handleViewChange}
        currentFilters={activeFilters}
      />

      {/* Toolbar */}
      <div className="flex items-center justify-between gap-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder={`Search ${objectType}...`}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant={showFilterPanel ? 'secondary' : 'outline'}
            size="sm"
            onClick={() => setShowFilterPanel((v) => !v)}
            data-testid="filter-toggle"
            className="gap-1.5"
          >
            <Filter className="h-4 w-4" />
            Filters
            {filterCount > 0 && (
              <Badge variant="default" className="ml-1 h-5 min-w-5 px-1.5" data-testid="filter-count-badge">
                {filterCount}
              </Badge>
            )}
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setColumnsDialogOpen(true)}
            data-testid="edit-columns-btn"
            className="gap-1.5"
          >
            <Columns className="h-4 w-4" />
            Columns
          </Button>
          <Button onClick={() => setCreateOpen(true)} className="gap-2">
            <Plus className="h-4 w-4" />
            Create
          </Button>
        </div>
      </div>

      {/* Filter Panel (collapsible) */}
      {showFilterPanel && (
        <FilterPanel
          properties={allFilterableProps}
          filterGroups={activeFilters}
          onFilterChange={handleFilterChange}
        />
      )}

      {/* Table or Empty State */}
      {rows.length === 0 && !isLoading ? (
        <EmptyState
          icon={Inbox}
          title={`No ${objectType} found`}
          description={
            useSearch
              ? 'Try adjusting your search query or filters.'
              : `Create your first ${singularize(objectType)} to get started.`
          }
          actionLabel={useSearch ? undefined : 'Create'}
          onAction={useSearch ? undefined : () => setCreateOpen(true)}
        />
      ) : (
        <>
          <DataTable
            columns={columns}
            data={rows}
            onRowClick={handleRowClick}
            loading={isLoading}
            rowKey={(row) => row.id as string}
            selectable
            selectedIds={selectedIds}
            onSelectionChange={setSelectedIds}
          />

          {!useSearch && (
            <Pagination
              hasNext={!!displayData?.paging?.next?.after}
              hasPrevious={afterStack.length > 0}
              onNext={handleNext}
              onPrevious={handlePrevious}
              pageLabel={`Page ${afterStack.length + 1}`}
            />
          )}
        </>
      )}

      {/* Create Dialog */}
      <ObjectCreateDialog
        objectType={objectType}
        open={createOpen}
        onOpenChange={setCreateOpen}
      />

      {/* Edit Columns Dialog */}
      <EditColumnsDialog
        open={columnsDialogOpen}
        onOpenChange={setColumnsDialogOpen}
        properties={allVisibleProps}
        selectedColumns={activeColumnNames}
        onSave={handleSaveColumns}
        onReset={handleResetColumns}
      />

      {/* Bulk Actions Bar */}
      <BulkActionsBar
        objectType={objectType}
        selectedIds={selectedIds}
        onClearSelection={() => setSelectedIds(new Set())}
        onComplete={() => {
          void queryClient.invalidateQueries({ queryKey: objectKeys.lists(objectType) })
        }}
      />
    </div>
  )
}
