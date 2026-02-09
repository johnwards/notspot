import { useState, useCallback } from 'react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useProperties } from '@/api/hooks/useProperties'
import { useUpdateObject, useArchiveObject } from '@/api/hooks/useObjects'
import { OwnerSelect } from '@/components/shared/OwnerSelect'
import { toast } from 'sonner'
import { Pencil, Trash2, X } from 'lucide-react'
import type { Property } from '@/api/types'

interface BulkActionsBarProps {
  objectType: string
  selectedIds: Set<string>
  onClearSelection: () => void
  onComplete: () => void
}

export function BulkActionsBar({
  objectType,
  selectedIds,
  onClearSelection,
  onComplete,
}: BulkActionsBarProps) {
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [selectedProperty, setSelectedProperty] = useState('')
  const [propertyValue, setPropertyValue] = useState('')
  const [progress, setProgress] = useState(0)
  const [total, setTotal] = useState(0)
  const [isProcessing, setIsProcessing] = useState(false)

  const { data: propertiesData } = useProperties(objectType)
  const updateMutation = useUpdateObject(objectType)
  const archiveMutation = useArchiveObject(objectType)

  const editableProperties = (propertiesData?.results ?? []).filter(
    (p: Property) => !p.hidden && !p.archived && !p.calculated,
  )

  const selectedProp = editableProperties.find(
    (p: Property) => p.name === selectedProperty,
  )

  const count = selectedIds.size

  const handleBulkEdit = useCallback(async () => {
    if (!selectedProperty || !propertyValue) return

    const ids = Array.from(selectedIds)
    setTotal(ids.length)
    setProgress(0)
    setIsProcessing(true)

    let successCount = 0
    let errorCount = 0

    for (const id of ids) {
      try {
        await new Promise<void>((resolve, reject) => {
          updateMutation.mutate(
            { id, properties: { [selectedProperty]: propertyValue } },
            {
              onSuccess: () => {
                successCount++
                resolve()
              },
              onError: (err) => {
                errorCount++
                reject(err)
              },
            },
          )
        })
      } catch {
        // continue on error
      }
      setProgress((prev) => prev + 1)
    }

    setIsProcessing(false)
    setEditOpen(false)
    setSelectedProperty('')
    setPropertyValue('')

    if (errorCount === 0) {
      toast.success(`Updated ${successCount} ${objectType}`)
    } else {
      toast.error(`Updated ${successCount}, failed ${errorCount}`)
    }
    onClearSelection()
    onComplete()
  }, [selectedIds, selectedProperty, propertyValue, updateMutation, objectType, onClearSelection, onComplete])

  const handleBulkDelete = useCallback(async () => {
    const ids = Array.from(selectedIds)
    setTotal(ids.length)
    setProgress(0)
    setIsProcessing(true)

    let successCount = 0
    let errorCount = 0

    for (const id of ids) {
      try {
        await new Promise<void>((resolve, reject) => {
          archiveMutation.mutate(id, {
            onSuccess: () => {
              successCount++
              resolve()
            },
            onError: (err) => {
              errorCount++
              reject(err)
            },
          })
        })
      } catch {
        // continue on error
      }
      setProgress((prev) => prev + 1)
    }

    setIsProcessing(false)
    setDeleteOpen(false)

    if (errorCount === 0) {
      toast.success(`Deleted ${successCount} ${objectType}`)
    } else {
      toast.error(`Deleted ${successCount}, failed ${errorCount}`)
    }
    onClearSelection()
    onComplete()
  }, [selectedIds, archiveMutation, objectType, onClearSelection, onComplete])

  if (count === 0) return null

  return (
    <>
      {/* Floating bar */}
      <div
        className="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 flex items-center gap-3 rounded-lg border bg-background px-4 py-3 shadow-lg"
        data-testid="bulk-actions-bar"
      >
        <span className="text-sm font-medium" data-testid="bulk-selection-count">
          {count} selected
        </span>
        <div className="h-4 w-px bg-border" />
        <Button
          variant="outline"
          size="sm"
          className="gap-1.5"
          onClick={() => setEditOpen(true)}
          data-testid="bulk-edit-btn"
        >
          <Pencil className="h-3.5 w-3.5" />
          Edit
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="gap-1.5 text-destructive hover:text-destructive"
          onClick={() => setDeleteOpen(true)}
          data-testid="bulk-delete-btn"
        >
          <Trash2 className="h-3.5 w-3.5" />
          Delete
        </Button>
        <Button
          variant="ghost"
          size="sm"
          className="gap-1.5"
          onClick={onClearSelection}
          data-testid="bulk-deselect-btn"
        >
          <X className="h-3.5 w-3.5" />
          Deselect all
        </Button>
        {isProcessing && (
          <>
            <div className="h-4 w-px bg-border" />
            <div className="flex items-center gap-2">
              <div className="h-2 w-24 rounded-full bg-muted overflow-hidden">
                <div
                  className="h-full bg-primary rounded-full transition-all"
                  style={{ width: `${total > 0 ? (progress / total) * 100 : 0}%` }}
                />
              </div>
              <span className="text-xs text-muted-foreground">
                {progress}/{total}
              </span>
            </div>
          </>
        )}
      </div>

      {/* Bulk Edit Dialog */}
      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Bulk Edit {count} {objectType}</DialogTitle>
            <DialogDescription>
              Select a property and value to apply to all selected records.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label>Property</Label>
              <Select value={selectedProperty} onValueChange={(val) => {
                setSelectedProperty(val)
                setPropertyValue('')
              }}>
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Choose a property..." />
                </SelectTrigger>
                <SelectContent>
                  {editableProperties.map((p: Property) => (
                    <SelectItem key={p.name} value={p.name}>
                      {p.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            {selectedProperty && (
              <div className="space-y-2">
                <Label>Value</Label>
                {selectedProperty === 'hubspot_owner_id' ? (
                  <OwnerSelect
                    value={propertyValue}
                    onValueChange={setPropertyValue}
                  />
                ) : selectedProp && (selectedProp.fieldType === 'select' || selectedProp.fieldType === 'radio') ? (
                  <Select value={propertyValue} onValueChange={setPropertyValue}>
                    <SelectTrigger className="w-full">
                      <SelectValue placeholder="Choose a value..." />
                    </SelectTrigger>
                    <SelectContent>
                      {selectedProp.options
                        ?.filter((o) => !o.hidden)
                        .sort((a, b) => a.displayOrder - b.displayOrder)
                        .map((opt) => (
                          <SelectItem key={opt.value} value={opt.value}>
                            {opt.label}
                          </SelectItem>
                        ))}
                    </SelectContent>
                  </Select>
                ) : (
                  <Input
                    value={propertyValue}
                    onChange={(e) => setPropertyValue(e.target.value)}
                    placeholder="Enter value..."
                    data-testid="bulk-edit-value"
                  />
                )}
              </div>
            )}
            <div className="flex justify-end gap-2 pt-2">
              <Button variant="outline" onClick={() => setEditOpen(false)}>
                Cancel
              </Button>
              <Button
                onClick={handleBulkEdit}
                disabled={!selectedProperty || !propertyValue || isProcessing}
                data-testid="bulk-edit-apply"
              >
                {isProcessing ? `Updating ${progress}/${total}...` : 'Apply'}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Bulk Delete Dialog */}
      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete {count} {objectType}?</DialogTitle>
            <DialogDescription>
              This action cannot be undone. All selected records will be archived.
            </DialogDescription>
          </DialogHeader>
          <div className="flex justify-end gap-2 pt-4">
            <Button variant="outline" onClick={() => setDeleteOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleBulkDelete}
              disabled={isProcessing}
              data-testid="bulk-delete-confirm"
            >
              {isProcessing ? `Deleting ${progress}/${total}...` : 'Delete'}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}
