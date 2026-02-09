import { useMemo } from 'react'
import { useAssociations } from '@/api/hooks/useAssociations'
import { useObject } from '@/api/hooks/useObjects'
import { ActivityTimelineItem } from '@/components/record/ActivityTimelineItem'
import { Skeleton } from '@/components/ui/skeleton'

type EngagementType = 'notes' | 'calls' | 'emails' | 'tasks' | 'meetings'

const ENGAGEMENT_TYPES: EngagementType[] = ['notes', 'calls', 'emails', 'tasks', 'meetings']

const ENGAGEMENT_PROPERTIES: Record<EngagementType, string[]> = {
  notes: ['hs_note_body'],
  calls: ['hs_call_body', 'hs_call_direction', 'hs_call_duration'],
  emails: ['hs_email_subject', 'hs_email_text'],
  tasks: ['hs_task_subject', 'hs_task_body', 'hs_task_status'],
  meetings: ['hs_meeting_title', 'hs_meeting_start_time', 'hs_meeting_end_time'],
}

interface ActivityTimelineProps {
  objectType: string
  objectId: string
}

function useEngagementTypeData(
  objectType: string,
  objectId: string,
  engagementType: EngagementType,
) {
  const { data: assocData, isLoading: assocLoading } = useAssociations(
    objectType,
    objectId,
    engagementType,
  )

  const associatedIds = useMemo(
    () => (assocData?.results ?? []).map((r) => r.toObjectId),
    [assocData],
  )

  return { associatedIds, isLoading: assocLoading }
}

/**
 * Component that fetches a single engagement and renders a timeline item.
 */
function EngagementItem({
  engagementType,
  engagementId,
}: {
  engagementType: EngagementType
  engagementId: string
}) {
  const properties = ENGAGEMENT_PROPERTIES[engagementType]
  const { data, isLoading } = useObject(engagementType, engagementId, properties)

  if (isLoading) {
    return (
      <div className="flex gap-3 pb-6">
        <Skeleton className="h-8 w-8 rounded-full shrink-0" />
        <div className="flex-1 space-y-2">
          <Skeleton className="h-4 w-24" />
          <Skeleton className="h-4 w-full" />
        </div>
      </div>
    )
  }

  if (!data) return null

  return <ActivityTimelineItem engagementType={engagementType} object={data} />
}

/**
 * Component that fetches associations for one engagement type and renders items.
 */
function EngagementTypeSection({
  objectType,
  objectId,
  engagementType,
}: {
  objectType: string
  objectId: string
  engagementType: EngagementType
}) {
  const { associatedIds, isLoading } = useEngagementTypeData(
    objectType,
    objectId,
    engagementType,
  )

  if (isLoading) {
    return (
      <div className="flex gap-3 pb-6">
        <Skeleton className="h-8 w-8 rounded-full shrink-0" />
        <div className="flex-1 space-y-2">
          <Skeleton className="h-4 w-24" />
          <Skeleton className="h-4 w-full" />
        </div>
      </div>
    )
  }

  return (
    <>
      {associatedIds.map((id) => (
        <EngagementItem
          key={`${engagementType}-${id}`}
          engagementType={engagementType}
          engagementId={id}
        />
      ))}
    </>
  )
}

export function ActivityTimeline({ objectType, objectId }: ActivityTimelineProps) {
  // Check if any associations exist to show empty state
  const notesAssoc = useAssociations(objectType, objectId, 'notes')
  const callsAssoc = useAssociations(objectType, objectId, 'calls')
  const emailsAssoc = useAssociations(objectType, objectId, 'emails')
  const tasksAssoc = useAssociations(objectType, objectId, 'tasks')
  const meetingsAssoc = useAssociations(objectType, objectId, 'meetings')

  const allLoading =
    notesAssoc.isLoading &&
    callsAssoc.isLoading &&
    emailsAssoc.isLoading &&
    tasksAssoc.isLoading &&
    meetingsAssoc.isLoading

  const totalAssociations =
    (notesAssoc.data?.results?.length ?? 0) +
    (callsAssoc.data?.results?.length ?? 0) +
    (emailsAssoc.data?.results?.length ?? 0) +
    (tasksAssoc.data?.results?.length ?? 0) +
    (meetingsAssoc.data?.results?.length ?? 0)

  const anyLoaded =
    !notesAssoc.isLoading ||
    !callsAssoc.isLoading ||
    !emailsAssoc.isLoading ||
    !tasksAssoc.isLoading ||
    !meetingsAssoc.isLoading

  if (allLoading) {
    return (
      <div className="space-y-4">
        {[1, 2, 3].map((i) => (
          <div key={i} className="flex gap-3 pb-6">
            <Skeleton className="h-8 w-8 rounded-full shrink-0" />
            <div className="flex-1 space-y-2">
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-4 w-full" />
            </div>
          </div>
        ))}
      </div>
    )
  }

  if (anyLoaded && totalAssociations === 0) {
    return (
      <div className="text-center py-12 text-muted-foreground" data-testid="timeline-empty">
        <p>No activities yet — use the action buttons to log your first activity</p>
      </div>
    )
  }

  return (
    <div data-testid="activity-timeline">
      {ENGAGEMENT_TYPES.map((type) => (
        <EngagementTypeSection
          key={type}
          objectType={objectType}
          objectId={objectId}
          engagementType={type}
        />
      ))}
    </div>
  )
}
