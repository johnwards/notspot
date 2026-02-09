import { useState, useCallback } from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { PropertyForm } from '@/components/objects/PropertyForm'
import { PropertyField } from '@/components/objects/PropertyField'
import { OwnerSelect } from '@/components/shared/OwnerSelect'
import { useUpdateObject } from '@/api/hooks/useObjects'
import { toast } from 'sonner'
import type { CrmObject, Property } from '@/api/types'
import { singularize } from '@/lib/utils'

interface AboutCardProps {
  objectType: string
  objectId: string
  object: CrmObject
  properties: Property[]
}

export function AboutCard({ objectType, objectId, object, properties }: AboutCardProps) {
  const [editingProp, setEditingProp] = useState<string | null>(null)
  const [editValue, setEditValue] = useState('')
  const [showAllProps, setShowAllProps] = useState(false)
  const updateMutation = useUpdateObject(objectType)

  // First 8 visible, non-calculated properties
  const visibleProps = properties
    .filter((p) => !p.hidden && !p.archived && !p.calculated)
    .sort((a, b) => a.displayOrder - b.displayOrder)
    .slice(0, 8)

  const handleStartEdit = (prop: Property) => {
    setEditingProp(prop.name)
    setEditValue(object.properties[prop.name] ?? '')
  }

  const handleSaveInline = useCallback(
    (propName: string, value: string) => {
      if (value === (object.properties[propName] ?? '')) {
        setEditingProp(null)
        return
      }
      updateMutation.mutate(
        { id: objectId, properties: { [propName]: value } },
        {
          onSuccess: () => {
            toast.success('Property updated')
            setEditingProp(null)
          },
          onError: (err) => {
            toast.error(`Failed to update: ${err.message}`)
          },
        },
      )
    },
    [objectId, object.properties, updateMutation],
  )

  const handleSaveAll = (changed: Record<string, string>) => {
    if (Object.keys(changed).length === 0) {
      setShowAllProps(false)
      return
    }
    updateMutation.mutate(
      { id: objectId, properties: changed },
      {
        onSuccess: () => {
          toast.success('Object updated')
          setShowAllProps(false)
        },
        onError: (err) => {
          toast.error(`Failed to update: ${err.message}`)
        },
      },
    )
  }

  const singular = singularize(objectType)
  const label = singular.charAt(0).toUpperCase() + singular.slice(1)

  return (
    <>
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">About this {label}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {visibleProps.map((prop) => {
            if (prop.name === 'hubspot_owner_id') {
              return (
                <div key={prop.name} className="space-y-1">
                  <p className="text-xs font-medium text-muted-foreground">{prop.label}</p>
                  <OwnerSelect
                    value={object.properties[prop.name] ?? ''}
                    onValueChange={(val) => handleSaveInline(prop.name, val)}
                    disabled={updateMutation.isPending}
                  />
                </div>
              )
            }
            return (
            <div key={prop.name} className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground">{prop.label}</p>
              {editingProp === prop.name ? (
                <div
                  onBlur={() => handleSaveInline(prop.name, editValue)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && prop.fieldType !== 'textarea') {
                      handleSaveInline(prop.name, editValue)
                    }
                    if (e.key === 'Escape') {
                      setEditingProp(null)
                    }
                  }}
                >
                  <PropertyField
                    property={prop}
                    value={editValue}
                    onChange={(_name, val) => setEditValue(val)}
                  />
                </div>
              ) : (
                <button
                  className="w-full text-left text-sm rounded px-1 py-0.5 hover:bg-muted transition-colors min-h-[24px]"
                  onClick={() => handleStartEdit(prop)}
                >
                  {object.properties[prop.name] || <span className="text-muted-foreground italic">—</span>}
                </button>
              )}
            </div>
            )
          })}

          <Button
            variant="link"
            size="sm"
            className="px-0 text-xs"
            onClick={() => setShowAllProps(true)}
          >
            See all properties
          </Button>
        </CardContent>
      </Card>

      <Dialog open={showAllProps} onOpenChange={setShowAllProps}>
        <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>All Properties</DialogTitle>
          </DialogHeader>
          <PropertyForm
            properties={properties}
            initialValues={object.properties}
            onSubmit={handleSaveAll}
            onCancel={() => setShowAllProps(false)}
            loading={updateMutation.isPending}
            submitLabel="Save Changes"
          />
        </DialogContent>
      </Dialog>
    </>
  )
}
