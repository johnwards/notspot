import { useCallback } from 'react'
import { Button } from '@/components/ui/button'
import { FilterRow } from './FilterRow'
import { Plus, Trash2 } from 'lucide-react'
import type { Filter, FilterGroup, Property } from '@/api/types'

interface FilterPanelProps {
  properties: Property[]
  filterGroups: FilterGroup[]
  onFilterChange: (groups: FilterGroup[]) => void
}

export function FilterPanel({ properties, filterGroups, onFilterChange }: FilterPanelProps) {
  // Work with the first filter group (single group for simplicity)
  const filters: Filter[] = filterGroups[0]?.filters ?? []

  const setFilters = useCallback(
    (newFilters: Filter[]) => {
      if (newFilters.length === 0) {
        onFilterChange([])
      } else {
        onFilterChange([{ filters: newFilters }])
      }
    },
    [onFilterChange],
  )

  const handleAdd = useCallback(() => {
    const firstProp = properties.find((p) => !p.hidden && !p.archived)
    setFilters([
      ...filters,
      {
        propertyName: firstProp?.name ?? '',
        operator: 'EQ',
        value: '',
      },
    ])
  }, [filters, properties, setFilters])

  const handleChange = useCallback(
    (index: number, updated: Filter) => {
      const next = [...filters]
      next[index] = updated
      setFilters(next)
    },
    [filters, setFilters],
  )

  const handleRemove = useCallback(
    (index: number) => {
      setFilters(filters.filter((_, i) => i !== index))
    },
    [filters, setFilters],
  )

  const handleClearAll = useCallback(() => {
    onFilterChange([])
  }, [onFilterChange])

  const filterableProperties = properties.filter((p) => !p.hidden && !p.archived)

  return (
    <div className="rounded-lg border bg-card p-4 space-y-3" data-testid="filter-panel">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium">Filters</h3>
        <div className="flex items-center gap-2">
          {filters.length > 0 && (
            <Button
              variant="ghost"
              size="xs"
              onClick={handleClearAll}
              className="text-muted-foreground"
              data-testid="clear-all-filters"
            >
              <Trash2 className="h-3 w-3" />
              Clear all
            </Button>
          )}
          <Button
            variant="outline"
            size="xs"
            onClick={handleAdd}
            data-testid="add-filter"
          >
            <Plus className="h-3 w-3" />
            Add filter
          </Button>
        </div>
      </div>

      {filters.length > 0 && (
        <div className="space-y-2">
          {filters.map((filter, index) => (
            <FilterRow
              key={index}
              properties={filterableProperties}
              filter={filter}
              onChange={(updated) => handleChange(index, updated)}
              onRemove={() => handleRemove(index)}
            />
          ))}
        </div>
      )}

      {filters.length === 0 && (
        <p className="text-sm text-muted-foreground">
          No filters applied. Click &quot;Add filter&quot; to narrow results.
        </p>
      )}
    </div>
  )
}
