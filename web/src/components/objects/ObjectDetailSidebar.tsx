import { useState, useMemo } from 'react'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from '@/components/ui/sheet'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import { PropertyForm } from './PropertyForm'
import { useObject, useUpdateObject, useArchiveObject } from '@/api/hooks/useObjects'
import { useProperties } from '@/api/hooks/useProperties'
import { toast } from 'sonner'
import { Trash2 } from 'lucide-react'
import { Separator } from '@/components/ui/separator'
import { AssociationPanel } from '@/components/associations/AssociationPanel'

interface ObjectDetailSidebarProps {
  objectType: string
  objectId: string | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ObjectDetailSidebar({
  objectType,
  objectId,
  open,
  onOpenChange,
}: ObjectDetailSidebarProps) {
  const [confirmArchive, setConfirmArchive] = useState(false)
  const { data: propertiesData } = useProperties(objectType)

  const properties = propertiesData?.results ?? []
  const propertyNames = useMemo(() => properties.map((p) => p.name), [properties])

  const { data: object, isLoading: objectLoading } = useObject(objectType, objectId ?? '', propertyNames)
  const updateMutation = useUpdateObject(objectType)
  const archiveMutation = useArchiveObject(objectType)

  const handleSave = (changed: Record<string, string>) => {
    if (!objectId || Object.keys(changed).length === 0) return
    updateMutation.mutate(
      { id: objectId, properties: changed },
      {
        onSuccess: () => {
          toast.success('Object updated')
        },
        onError: (err) => {
          toast.error(`Failed to update: ${err.message}`)
        },
      },
    )
  }

  const handleArchive = () => {
    if (!objectId) return
    archiveMutation.mutate(objectId, {
      onSuccess: () => {
        toast.success('Object archived')
        setConfirmArchive(false)
        onOpenChange(false)
      },
      onError: (err) => {
        toast.error(`Failed to archive: ${err.message}`)
      },
    })
  }

  return (
    <>
      <Sheet open={open} onOpenChange={onOpenChange}>
        <SheetContent side="right" className="w-full sm:max-w-lg flex flex-col overflow-hidden">
          <SheetHeader>
            <SheetTitle>
              {objectType.charAt(0).toUpperCase() + objectType.slice(1)} Details
            </SheetTitle>
            <SheetDescription>ID: {objectId}</SheetDescription>
          </SheetHeader>

          {objectLoading ? (
            <div className="space-y-4 p-4">
              <Skeleton className="h-4 w-1/3" />
              <Skeleton className="h-9 w-full" />
              <Skeleton className="h-4 w-1/4" />
              <Skeleton className="h-9 w-full" />
              <Skeleton className="h-4 w-2/5" />
              <Skeleton className="h-9 w-full" />
              <Skeleton className="h-4 w-1/3" />
              <Skeleton className="h-9 w-full" />
            </div>
          ) : object ? (
            <ScrollArea className="flex-1 px-4 overflow-auto">
              <PropertyForm
                properties={properties}
                initialValues={object.properties}
                onSubmit={handleSave}
                loading={updateMutation.isPending}
                submitLabel="Save Changes"
              />

              <div className="mt-6 border-t pt-4">
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => setConfirmArchive(true)}
                  className="gap-2"
                >
                  <Trash2 className="h-4 w-4" />
                  Archive
                </Button>
              </div>

              {objectId && (
                <>
                  <Separator className="my-6" />
                  <AssociationPanel objectType={objectType} objectId={objectId} />
                </>
              )}
            </ScrollArea>
          ) : (
            <div className="p-4 text-sm text-muted-foreground">Object not found</div>
          )}
        </SheetContent>
      </Sheet>

      <Dialog open={confirmArchive} onOpenChange={setConfirmArchive}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Archive this object?</DialogTitle>
            <DialogDescription>
              This will archive the object. It can be restored later.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirmArchive(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleArchive}
              disabled={archiveMutation.isPending}
            >
              {archiveMutation.isPending ? 'Archiving...' : 'Archive'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
