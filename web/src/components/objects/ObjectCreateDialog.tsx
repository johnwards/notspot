import { useState, useCallback, useMemo } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Button } from '@/components/ui/button'
import { PropertyForm } from './PropertyForm'
import { AssociationSearchDialog } from '@/components/associations/AssociationSearchDialog'
import { useCreateObject } from '@/api/hooks/useObjects'
import { useProperties } from '@/api/hooks/useProperties'
import { apiFetch } from '@/api/client'
import { toast } from 'sonner'
import { Plus, X } from 'lucide-react'

const ASSOCIATION_TARGETS: Record<string, string[]> = {
  deals: ['contacts', 'companies'],
  tickets: ['contacts', 'companies'],
  contacts: ['companies'],
}

interface PendingAssociation {
  toType: string
  toId: string
  displayValue: string
}

interface ObjectCreateDialogProps {
  objectType: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ObjectCreateDialog({
  objectType,
  open,
  onOpenChange,
}: ObjectCreateDialogProps) {
  const { data: propertiesData } = useProperties(objectType)
  const createMutation = useCreateObject(objectType)
  const [pendingAssociations, setPendingAssociations] = useState<PendingAssociation[]>([])
  const [assocSearchType, setAssocSearchType] = useState<string | null>(null)

  const properties = propertiesData?.results ?? []
  const associationTargets = ASSOCIATION_TARGETS[objectType] ?? []
  const emptyInitialValues = useMemo<Record<string, string>>(() => ({}), [])

  const handleClose = useCallback((isOpen: boolean) => {
    if (!isOpen) {
      setPendingAssociations([])
      setAssocSearchType(null)
    }
    onOpenChange(isOpen)
  }, [onOpenChange])

  const handleSubmit = (values: Record<string, string>) => {
    if (Object.keys(values).length === 0) return
    createMutation.mutate(
      { properties: values },
      {
        onSuccess: (created) => {
          // Fire association PUTs for each pending association
          const assocPromises = pendingAssociations.map((assoc) =>
            apiFetch<void>(
              `/crm/v4/objects/${objectType}/${created.id}/associations/default/${assoc.toType}/${assoc.toId}`,
              { method: 'PUT', body: JSON.stringify({}) },
            ).catch(() => {
              // Silently ignore individual association failures
            }),
          )
          void Promise.all(assocPromises).then(() => {
            const assocCount = pendingAssociations.length
            const msg = assocCount > 0
              ? `${objectType.slice(0, -1)} created with ${assocCount} association${assocCount > 1 ? 's' : ''}`
              : `${objectType.slice(0, -1)} created`
            toast.success(msg)
            setPendingAssociations([])
            handleClose(false)
          })
        },
        onError: (err) => {
          toast.error(`Failed to create: ${err.message}`)
        },
      },
    )
  }

  const handleRemoveAssociation = (index: number) => {
    setPendingAssociations((prev) => prev.filter((_, i) => i !== index))
  }

  return (
    <>
      <Dialog open={open} onOpenChange={handleClose}>
        <DialogContent className="max-h-[80vh] sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>
              Create {objectType.endsWith('ies')
                ? objectType.charAt(0).toUpperCase() + objectType.slice(1, -3) + 'y'
                : objectType.charAt(0).toUpperCase() + objectType.slice(1, -1)}
            </DialogTitle>
            <DialogDescription>Fill in the properties below.</DialogDescription>
          </DialogHeader>

          <ScrollArea className="max-h-[60vh] pr-4">
            <PropertyForm
              properties={properties}
              initialValues={emptyInitialValues}
              onSubmit={handleSubmit}
              onCancel={() => handleClose(false)}
              submitLabel="Create"
              loading={createMutation.isPending}
              createMode
            />

            {associationTargets.length > 0 && (
              <div className="mt-4 space-y-3 border-t pt-4">
                <p className="text-sm font-medium text-muted-foreground">Associate with...</p>
                {associationTargets.map((targetType) => {
                  const label = targetType.charAt(0).toUpperCase() + targetType.slice(1)
                  const assocs = pendingAssociations.filter((a) => a.toType === targetType)
                  return (
                    <div key={targetType} className="space-y-2">
                      <div className="flex items-center justify-between">
                        <span className="text-sm">{label}</span>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="gap-1.5 h-7"
                          onClick={() => setAssocSearchType(targetType)}
                          data-testid={`create-assoc-add-${targetType}`}
                        >
                          <Plus className="h-3.5 w-3.5" />
                          Add
                        </Button>
                      </div>
                      {assocs.map((assoc, i) => {
                        const globalIndex = pendingAssociations.indexOf(assoc)
                        return (
                          <div
                            key={`${assoc.toType}-${assoc.toId}`}
                            className="flex items-center justify-between rounded-md bg-muted px-3 py-1.5 text-sm"
                          >
                            <span className="truncate">{assoc.displayValue}</span>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-6 w-6 p-0"
                              onClick={() => handleRemoveAssociation(globalIndex)}
                              data-testid={`create-assoc-remove-${i}`}
                            >
                              <X className="h-3.5 w-3.5" />
                            </Button>
                          </div>
                        )
                      })}
                    </div>
                  )
                })}
              </div>
            )}
          </ScrollArea>
        </DialogContent>
      </Dialog>

      {assocSearchType && (
        <AssociationSearchDialog
          open={assocSearchType !== null}
          onOpenChange={(isOpen) => {
            if (!isOpen) setAssocSearchType(null)
          }}
          fromType={objectType}
          fromId=""
          toType={assocSearchType}
          onSelect={(record) => {
            setPendingAssociations((prev) => {
              // Avoid duplicates
              if (prev.some((a) => a.toType === assocSearchType && a.toId === record.id)) {
                return prev
              }
              return [...prev, { toType: assocSearchType!, toId: record.id, displayValue: record.displayValue }]
            })
            setAssocSearchType(null)
          }}
        />
      )}
    </>
  )
}
