import { useState } from 'react'
import { StickyNote, Phone, Mail, CheckSquare, Calendar } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { EngagementCreateDialog } from '@/components/record/EngagementCreateDialog'

type EngagementType = 'notes' | 'calls' | 'emails' | 'tasks' | 'meetings'

interface ActionButtonsProps {
  objectType: string
  objectId: string
}

const ACTIONS: { type: EngagementType; label: string; icon: typeof StickyNote }[] = [
  { type: 'notes', label: 'Note', icon: StickyNote },
  { type: 'calls', label: 'Call', icon: Phone },
  { type: 'emails', label: 'Email', icon: Mail },
  { type: 'tasks', label: 'Task', icon: CheckSquare },
  { type: 'meetings', label: 'Meeting', icon: Calendar },
]

export function ActionButtons({ objectType, objectId }: ActionButtonsProps) {
  const [openType, setOpenType] = useState<EngagementType | null>(null)

  return (
    <>
      <div className="grid grid-cols-2 gap-2" data-testid="action-buttons">
        {ACTIONS.map(({ type, label, icon: Icon }) => (
          <Button
            key={type}
            variant="outline"
            size="sm"
            className="justify-start gap-2"
            onClick={() => setOpenType(type)}
            data-testid={`action-btn-${type}`}
          >
            <Icon className="h-4 w-4" />
            {label}
          </Button>
        ))}
      </div>

      {openType && (
        <EngagementCreateDialog
          objectType={objectType}
          objectId={objectId}
          engagementType={openType}
          open={true}
          onOpenChange={(open) => {
            if (!open) setOpenType(null)
          }}
        />
      )}
    </>
  )
}
