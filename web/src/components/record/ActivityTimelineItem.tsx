import { StickyNote, Phone, Mail, CheckSquare, Calendar } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import type { CrmObject } from '@/api/types'

type EngagementType = 'notes' | 'calls' | 'emails' | 'tasks' | 'meetings'

interface ActivityTimelineItemProps {
  engagementType: EngagementType
  object: CrmObject
}

const ICON_MAP: Record<EngagementType, typeof StickyNote> = {
  notes: StickyNote,
  calls: Phone,
  emails: Mail,
  tasks: CheckSquare,
  meetings: Calendar,
}

const COLOR_MAP: Record<EngagementType, string> = {
  notes: 'bg-yellow-500',
  calls: 'bg-blue-500',
  emails: 'bg-green-500',
  tasks: 'bg-purple-500',
  meetings: 'bg-orange-500',
}

const TYPE_LABELS: Record<EngagementType, string> = {
  notes: 'Note',
  calls: 'Call',
  emails: 'Email',
  tasks: 'Task',
  meetings: 'Meeting',
}

function getBodyText(type: EngagementType, props: Record<string, string>): string {
  switch (type) {
    case 'notes':
      return props.hs_note_body ?? ''
    case 'calls':
      return props.hs_call_body ?? ''
    case 'emails':
      return props.hs_email_subject
        ? `${props.hs_email_subject}${props.hs_email_text ? ` - ${props.hs_email_text}` : ''}`
        : props.hs_email_text ?? ''
    case 'tasks':
      return props.hs_task_subject
        ? `${props.hs_task_subject}${props.hs_task_body ? ` - ${props.hs_task_body}` : ''}`
        : props.hs_task_body ?? ''
    case 'meetings':
      return props.hs_meeting_title ?? ''
  }
}

function formatRelativeTime(dateStr: string): string {
  try {
    const date = new Date(dateStr)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffMinutes = Math.floor(diffMs / (1000 * 60))
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60))
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))

    if (diffMinutes < 1) return 'just now'
    if (diffMinutes < 60) return `${diffMinutes}m ago`
    if (diffHours < 24) return `${diffHours}h ago`
    if (diffDays < 30) return `${diffDays}d ago`
    return date.toLocaleDateString()
  } catch {
    return ''
  }
}

export function ActivityTimelineItem({ engagementType, object }: ActivityTimelineItemProps) {
  const Icon = ICON_MAP[engagementType]
  const colorClass = COLOR_MAP[engagementType]
  const body = getBodyText(engagementType, object.properties)
  const timestamp = formatRelativeTime(object.createdAt)

  return (
    <div className="relative flex gap-3 pb-6" data-testid={`timeline-item-${engagementType}`}>
      {/* Timeline dot and line */}
      <div className="flex flex-col items-center">
        <div
          className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-full ${colorClass} text-white`}
          data-testid={`timeline-icon-${engagementType}`}
        >
          <Icon className="h-4 w-4" />
        </div>
        <div className="w-px flex-1 bg-border" />
      </div>

      {/* Content */}
      <div className="flex-1 min-w-0 pt-0.5">
        <div className="flex items-center gap-2 mb-1">
          <Badge variant="secondary" className="text-xs">
            {TYPE_LABELS[engagementType]}
          </Badge>
          {timestamp && (
            <span className="text-xs text-muted-foreground">{timestamp}</span>
          )}
        </div>
        {body && (
          <p className="text-sm text-foreground line-clamp-3">{body}</p>
        )}
      </div>
    </div>
  )
}
