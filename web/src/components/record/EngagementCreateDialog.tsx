import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { apiFetch } from '@/api/client'
import { associationKeys } from '@/api/hooks/useAssociations'
import { toast } from 'sonner'
import type { CrmObject } from '@/api/types'

type EngagementType = 'notes' | 'calls' | 'emails' | 'tasks' | 'meetings'

interface EngagementCreateDialogProps {
  objectType: string
  objectId: string
  engagementType: EngagementType
  open: boolean
  onOpenChange: (open: boolean) => void
}

const ENGAGEMENT_LABELS: Record<EngagementType, string> = {
  notes: 'Note',
  calls: 'Call',
  emails: 'Email',
  tasks: 'Task',
  meetings: 'Meeting',
}

export function EngagementCreateDialog({
  objectType,
  objectId,
  engagementType,
  open,
  onOpenChange,
}: EngagementCreateDialogProps) {
  const queryClient = useQueryClient()
  const [submitting, setSubmitting] = useState(false)
  const [formData, setFormData] = useState<Record<string, string>>({})

  const handleChange = (field: string, value: string) => {
    setFormData((prev) => ({ ...prev, [field]: value }))
  }

  const handleSubmit = async () => {
    setSubmitting(true)
    try {
      // 1. Create engagement
      const created = await apiFetch<CrmObject>(
        `/crm/v3/objects/${engagementType}`,
        {
          method: 'POST',
          body: JSON.stringify({ properties: formData }),
        },
      )

      // 2. Associate with parent object
      await apiFetch<void>(
        `/crm/v4/objects/${engagementType}/${created.id}/associations/default/${objectType}/${objectId}`,
        { method: 'PUT' },
      )

      // 3. Invalidate association queries
      await queryClient.invalidateQueries({
        queryKey: associationKeys.list(objectType, objectId, engagementType),
      })

      // 4. Close and notify
      toast.success(`${ENGAGEMENT_LABELS[engagementType]} created`)
      setFormData({})
      onOpenChange(false)
    } catch (err) {
      toast.error(
        `Failed to create ${ENGAGEMENT_LABELS[engagementType].toLowerCase()}: ${err instanceof Error ? err.message : 'Unknown error'}`,
      )
    } finally {
      setSubmitting(false)
    }
  }

  const renderFields = () => {
    switch (engagementType) {
      case 'notes':
        return (
          <div className="space-y-2">
            <Label htmlFor="hs_note_body">Note Body</Label>
            <textarea
              id="hs_note_body"
              className="border-input bg-transparent flex min-h-[80px] w-full rounded-md border px-3 py-2 text-sm shadow-xs focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] outline-none"
              placeholder="Enter note..."
              value={formData.hs_note_body ?? ''}
              onChange={(e) => handleChange('hs_note_body', e.target.value)}
            />
          </div>
        )
      case 'calls':
        return (
          <>
            <div className="space-y-2">
              <Label htmlFor="hs_call_body">Call Body</Label>
              <textarea
                id="hs_call_body"
                className="border-input bg-transparent flex min-h-[80px] w-full rounded-md border px-3 py-2 text-sm shadow-xs focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] outline-none"
                placeholder="Call notes..."
                value={formData.hs_call_body ?? ''}
                onChange={(e) => handleChange('hs_call_body', e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="hs_call_direction">Direction</Label>
              <Select
                value={formData.hs_call_direction ?? ''}
                onValueChange={(v) => handleChange('hs_call_direction', v)}
              >
                <SelectTrigger id="hs_call_direction" className="w-full">
                  <SelectValue placeholder="Select direction" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="INBOUND">Inbound</SelectItem>
                  <SelectItem value="OUTBOUND">Outbound</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="hs_call_duration">Duration (seconds)</Label>
              <Input
                id="hs_call_duration"
                type="number"
                placeholder="0"
                value={formData.hs_call_duration ?? ''}
                onChange={(e) => handleChange('hs_call_duration', e.target.value)}
              />
            </div>
          </>
        )
      case 'emails':
        return (
          <>
            <div className="space-y-2">
              <Label htmlFor="hs_email_subject">Subject</Label>
              <Input
                id="hs_email_subject"
                placeholder="Email subject..."
                value={formData.hs_email_subject ?? ''}
                onChange={(e) => handleChange('hs_email_subject', e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="hs_email_text">Body</Label>
              <textarea
                id="hs_email_text"
                className="border-input bg-transparent flex min-h-[80px] w-full rounded-md border px-3 py-2 text-sm shadow-xs focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] outline-none"
                placeholder="Email body..."
                value={formData.hs_email_text ?? ''}
                onChange={(e) => handleChange('hs_email_text', e.target.value)}
              />
            </div>
          </>
        )
      case 'tasks':
        return (
          <>
            <div className="space-y-2">
              <Label htmlFor="hs_task_subject">Subject</Label>
              <Input
                id="hs_task_subject"
                placeholder="Task subject..."
                value={formData.hs_task_subject ?? ''}
                onChange={(e) => handleChange('hs_task_subject', e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="hs_task_body">Body</Label>
              <textarea
                id="hs_task_body"
                className="border-input bg-transparent flex min-h-[80px] w-full rounded-md border px-3 py-2 text-sm shadow-xs focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] outline-none"
                placeholder="Task details..."
                value={formData.hs_task_body ?? ''}
                onChange={(e) => handleChange('hs_task_body', e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="hs_task_status">Status</Label>
              <Select
                value={formData.hs_task_status ?? ''}
                onValueChange={(v) => handleChange('hs_task_status', v)}
              >
                <SelectTrigger id="hs_task_status" className="w-full">
                  <SelectValue placeholder="Select status" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="NOT_STARTED">Not Started</SelectItem>
                  <SelectItem value="IN_PROGRESS">In Progress</SelectItem>
                  <SelectItem value="COMPLETED">Completed</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </>
        )
      case 'meetings':
        return (
          <>
            <div className="space-y-2">
              <Label htmlFor="hs_meeting_title">Title</Label>
              <Input
                id="hs_meeting_title"
                placeholder="Meeting title..."
                value={formData.hs_meeting_title ?? ''}
                onChange={(e) => handleChange('hs_meeting_title', e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="hs_meeting_start_time">Start Time</Label>
              <Input
                id="hs_meeting_start_time"
                type="datetime-local"
                value={formData.hs_meeting_start_time ?? ''}
                onChange={(e) =>
                  handleChange('hs_meeting_start_time', e.target.value)
                }
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="hs_meeting_end_time">End Time</Label>
              <Input
                id="hs_meeting_end_time"
                type="datetime-local"
                value={formData.hs_meeting_end_time ?? ''}
                onChange={(e) =>
                  handleChange('hs_meeting_end_time', e.target.value)
                }
              />
            </div>
          </>
        )
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            Create {ENGAGEMENT_LABELS[engagementType]}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-2">{renderFields()}</div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={submitting}>
            {submitting ? 'Creating...' : 'Create'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
