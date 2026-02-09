import { useMemo } from 'react'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { X } from 'lucide-react'
import type { Filter, Property } from '@/api/types'

interface FilterRowProps {
  properties: Property[]
  filter: Filter
  onChange: (filter: Filter) => void
  onRemove: () => void
}

const OPERATORS_BY_TYPE: Record<string, { value: string; label: string }[]> = {
  string: [
    { value: 'EQ', label: 'is equal to' },
    { value: 'NEQ', label: 'is not equal to' },
    { value: 'CONTAINS_TOKEN', label: 'contains' },
    { value: 'HAS_PROPERTY', label: 'is known' },
    { value: 'NOT_HAS_PROPERTY', label: 'is unknown' },
  ],
  number: [
    { value: 'EQ', label: 'is equal to' },
    { value: 'NEQ', label: 'is not equal to' },
    { value: 'LT', label: 'is less than' },
    { value: 'LTE', label: 'is less than or equal to' },
    { value: 'GT', label: 'is greater than' },
    { value: 'GTE', label: 'is greater than or equal to' },
    { value: 'BETWEEN', label: 'is between' },
  ],
  date: [
    { value: 'EQ', label: 'is equal to' },
    { value: 'GT', label: 'is after' },
    { value: 'GTE', label: 'is after or on' },
    { value: 'LT', label: 'is before' },
    { value: 'LTE', label: 'is before or on' },
    { value: 'BETWEEN', label: 'is between' },
  ],
  datetime: [
    { value: 'EQ', label: 'is equal to' },
    { value: 'GT', label: 'is after' },
    { value: 'GTE', label: 'is after or on' },
    { value: 'LT', label: 'is before' },
    { value: 'LTE', label: 'is before or on' },
    { value: 'BETWEEN', label: 'is between' },
  ],
  enumeration: [
    { value: 'EQ', label: 'is equal to' },
    { value: 'NEQ', label: 'is not equal to' },
    { value: 'IN', label: 'is any of' },
  ],
}

function getOperatorsForType(type: string): { value: string; label: string }[] {
  return OPERATORS_BY_TYPE[type] || OPERATORS_BY_TYPE.string
}

const NO_VALUE_OPERATORS = new Set(['HAS_PROPERTY', 'NOT_HAS_PROPERTY'])

export function FilterRow({ properties, filter, onChange, onRemove }: FilterRowProps) {
  const selectedProperty = useMemo(
    () => properties.find((p) => p.name === filter.propertyName),
    [properties, filter.propertyName],
  )

  const operators = useMemo(
    () => getOperatorsForType(selectedProperty?.type ?? 'string'),
    [selectedProperty],
  )

  const hideValue = NO_VALUE_OPERATORS.has(filter.operator)

  return (
    <div className="flex items-center gap-2" data-testid="filter-row">
      {/* Property select */}
      <Select
        value={filter.propertyName}
        onValueChange={(value) => {
          const prop = properties.find((p) => p.name === value)
          const newOps = getOperatorsForType(prop?.type ?? 'string')
          onChange({
            propertyName: value,
            operator: newOps[0]?.value ?? 'EQ',
            value: '',
          })
        }}
      >
        <SelectTrigger className="w-[180px]" aria-label="Property">
          <SelectValue placeholder="Select property" />
        </SelectTrigger>
        <SelectContent>
          {properties.map((p) => (
            <SelectItem key={p.name} value={p.name}>
              {p.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      {/* Operator select */}
      <Select
        value={filter.operator}
        onValueChange={(value) =>
          onChange({
            ...filter,
            operator: value,
            ...(NO_VALUE_OPERATORS.has(value) ? { value: undefined } : {}),
          })
        }
      >
        <SelectTrigger className="w-[180px]" aria-label="Operator">
          <SelectValue placeholder="Select operator" />
        </SelectTrigger>
        <SelectContent>
          {operators.map((op) => (
            <SelectItem key={op.value} value={op.value}>
              {op.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      {/* Value input */}
      {!hideValue && (
        <Input
          placeholder="Enter value"
          value={filter.value ?? ''}
          onChange={(e) => onChange({ ...filter, value: e.target.value })}
          className="w-[200px]"
          data-testid="filter-value"
        />
      )}

      {/* Remove button */}
      <Button
        variant="ghost"
        size="icon-xs"
        onClick={onRemove}
        aria-label="Remove filter"
      >
        <X className="h-3 w-3" />
      </Button>
    </div>
  )
}
