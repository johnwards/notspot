import { useMemo, useState } from 'react'
import { useAssociations } from '@/api/hooks/useAssociations'
import { useRemoveAssociation } from '@/api/hooks/useAssociationMutations'
import { useSchemas } from '@/api/hooks/useSchemas'
import { useObject } from '@/api/hooks/useObjects'
import { AssociationCard } from './AssociationCard'
import { AssociationSearchDialog } from './AssociationSearchDialog'
import { Loader2 } from 'lucide-react'
import { toast } from 'sonner'

const ALL_TYPES = ['contacts', 'companies', 'deals', 'tickets'] as const

const DISPLAY_PROPERTY_FALLBACKS: Record<string, string> = {
  contacts: 'email',
  companies: 'name',
  deals: 'dealname',
  tickets: 'subject',
}

interface AssociationPanelProps {
  objectType: string
  objectId: string
}

function AssociationGroup({
  fromType,
  objectId,
  toType,
  displayProperty,
  onAdd,
}: {
  fromType: string
  objectId: string
  toType: string
  displayProperty: string
  onAdd: () => void
}) {
  const { data, isLoading } = useAssociations(fromType, objectId, toType)
  const removeMutation = useRemoveAssociation(fromType, objectId, toType)

  const associationResults = data?.results ?? []

  const handleRemove = (toId: string) => {
    removeMutation.mutate(toId, {
      onSuccess: () => {
        toast.success('Association removed')
      },
      onError: (err) => {
        toast.error(`Failed to remove: ${err.message}`)
      },
    })
  }

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 py-2 text-sm text-muted-foreground">
        <Loader2 className="h-3 w-3 animate-spin" />
        Loading {toType}...
      </div>
    )
  }

  if (associationResults.length === 0) {
    return <AssociationCard objectType={toType} items={[]} onAdd={onAdd} onRemove={handleRemove} />
  }

  return (
    <AssociationGroupWithDetails
      fromType={fromType}
      objectId={objectId}
      toType={toType}
      objectIds={associationResults.map((r) => r.toObjectId)}
      displayProperty={displayProperty}
      onAdd={onAdd}
      onRemove={handleRemove}
    />
  )
}

function AssociationGroupWithDetails({
  fromType,
  objectId,
  toType,
  objectIds,
  displayProperty,
  onAdd,
  onRemove,
}: {
  fromType: string
  objectId: string
  toType: string
  objectIds: string[]
  displayProperty: string
  onAdd: () => void
  onRemove: (id: string) => void
}) {
  const ids = objectIds.slice(0, 5)

  return (
    <AssociationCardWithFetch
      fromType={fromType}
      objectId={objectId}
      toType={toType}
      objectIds={ids}
      totalCount={objectIds.length}
      displayProperty={displayProperty}
      onAdd={onAdd}
      onRemove={onRemove}
    />
  )
}

function AssociationCardWithFetch({
  toType,
  objectIds,
  totalCount,
  displayProperty,
  onAdd,
  onRemove,
}: {
  fromType: string
  objectId: string
  toType: string
  objectIds: string[]
  totalCount: number
  displayProperty: string
  onAdd: () => void
  onRemove: (id: string) => void
}) {
  const props = [displayProperty]
  const obj0 = useObject(toType, objectIds[0] ?? '', props)
  const obj1 = useObject(toType, objectIds[1] ?? '', props)
  const obj2 = useObject(toType, objectIds[2] ?? '', props)
  const obj3 = useObject(toType, objectIds[3] ?? '', props)
  const obj4 = useObject(toType, objectIds[4] ?? '', props)

  const items = useMemo(() => {
    const fetched = [obj0, obj1, obj2, obj3, obj4]
    return objectIds.map((id, i) => {
      const obj = fetched[i]?.data
      const displayValue = obj?.properties[displayProperty] || `${toType} #${id}`
      return { id, displayValue }
    })
  }, [objectIds, obj0, obj1, obj2, obj3, obj4, displayProperty, toType])

  const displayItems =
    totalCount > 5
      ? [...items, { id: '__more', displayValue: `+${totalCount - 5} more...` }]
      : items

  return <AssociationCard objectType={toType} items={displayItems} onAdd={onAdd} onRemove={onRemove} />
}

export function AssociationPanel({ objectType, objectId }: AssociationPanelProps) {
  const { data: schemasData } = useSchemas()
  const [addDialogType, setAddDialogType] = useState<string | null>(null)

  const displayPropertyMap = useMemo(() => {
    const map: Record<string, string> = { ...DISPLAY_PROPERTY_FALLBACKS }
    if (schemasData?.results) {
      for (const schema of schemasData.results) {
        if (schema.primaryDisplayProperty) {
          map[schema.name] = schema.primaryDisplayProperty
        }
      }
    }
    return map
  }, [schemasData])

  const targetTypes = useMemo(
    () => ALL_TYPES.filter((t) => t !== objectType),
    [objectType],
  )

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-medium text-muted-foreground">Associations</h3>
      {targetTypes.map((toType) => (
        <AssociationGroup
          key={toType}
          fromType={objectType}
          objectId={objectId}
          toType={toType}
          displayProperty={displayPropertyMap[toType] ?? 'name'}
          onAdd={() => setAddDialogType(toType)}
        />
      ))}

      {addDialogType !== null && (
        <AssociationSearchDialog
          open
          onOpenChange={(open) => {
            if (!open) setAddDialogType(null)
          }}
          fromType={objectType}
          fromId={objectId}
          toType={addDialogType}
        />
      )}
    </div>
  )
}
