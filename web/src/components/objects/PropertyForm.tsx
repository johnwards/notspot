import { useState, useCallback, useEffect } from 'react'
import { PropertyField } from './PropertyField'
import { Button } from '@/components/ui/button'
import type { Property } from '@/api/types'

interface PropertyFormProps {
  properties: Property[]
  initialValues: Record<string, string>
  onSubmit: (values: Record<string, string>) => void
  onCancel?: () => void
  submitLabel?: string
  loading?: boolean
  /** If true, only show required/common fields (for create dialogs) */
  createMode?: boolean
}

export function PropertyForm({
  properties,
  initialValues,
  onSubmit,
  onCancel,
  submitLabel = 'Save',
  loading,
  createMode,
}: PropertyFormProps) {
  const [values, setValues] = useState<Record<string, string>>(initialValues)

  // Sync state when initialValues change (e.g., when object data loads)
  useEffect(() => {
    setValues(initialValues)
  }, [initialValues])

  const handleChange = useCallback((name: string, value: string) => {
    setValues((prev) => ({ ...prev, [name]: value }))
  }, [])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    // Only return changed values
    const changed: Record<string, string> = {}
    for (const [key, val] of Object.entries(values)) {
      if (val !== (initialValues[key] ?? '')) {
        changed[key] = val
      }
    }
    onSubmit(changed)
  }

  // Filter properties
  const visibleProps = properties
    .filter((p) => !p.archived && !p.hidden)
    .filter((p) => {
      if (createMode) {
        // In create mode show editable properties (not calculated/system)
        return !p.calculated
      }
      return true
    })
    .sort((a, b) => a.displayOrder - b.displayOrder)

  // Group by groupName
  const groups = new Map<string, Property[]>()
  for (const prop of visibleProps) {
    const group = prop.groupName || 'Other'
    if (!groups.has(group)) {
      groups.set(group, [])
    }
    groups.get(group)!.push(prop)
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {Array.from(groups.entries()).map(([groupName, groupProps]) => (
        <div key={groupName} className="space-y-4">
          <h4 className="text-sm font-semibold capitalize text-muted-foreground">
            {groupName.replace(/_/g, ' ')}
          </h4>
          {groupProps.map((prop) => (
            <PropertyField
              key={prop.name}
              property={prop}
              value={values[prop.name] ?? ''}
              onChange={handleChange}
            />
          ))}
        </div>
      ))}

      <div className="flex justify-end gap-2 pt-4">
        {onCancel && (
          <Button type="button" variant="outline" onClick={onCancel}>
            Cancel
          </Button>
        )}
        <Button type="submit" disabled={loading}>
          {loading ? 'Saving...' : submitLabel}
        </Button>
      </div>
    </form>
  )
}
