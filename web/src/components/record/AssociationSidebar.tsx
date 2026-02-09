import { ScrollArea } from '@/components/ui/scroll-area'
import { AssociationPanel } from '@/components/associations/AssociationPanel'

interface AssociationSidebarProps {
  objectType: string
  objectId: string
}

export function AssociationSidebar({ objectType, objectId }: AssociationSidebarProps) {
  return (
    <div className="space-y-3">
      <ScrollArea className="h-[calc(100vh-200px)]">
        <AssociationPanel objectType={objectType} objectId={objectId} />
      </ScrollArea>
    </div>
  )
}
