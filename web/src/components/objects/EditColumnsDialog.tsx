import { useState, useEffect } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import type { Property } from '@/api/types'

interface EditColumnsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  properties: Property[]
  selectedColumns: string[]
  onSave: (columns: string[]) => void
  onReset: () => void
}

export function EditColumnsDialog({
  open,
  onOpenChange,
  properties,
  selectedColumns,
  onSave,
  onReset,
}: EditColumnsDialogProps) {
  const [checked, setChecked] = useState<Set<string>>(new Set(selectedColumns))

  // Sync when dialog opens or selectedColumns change
  useEffect(() => {
    if (open) {
      setChecked(new Set(selectedColumns))
    }
  }, [open, selectedColumns])

  const availableProps = properties
    .filter((p) => !p.hidden && !p.archived)
    .sort((a, b) => a.displayOrder - b.displayOrder)

  const handleToggle = (name: string) => {
    setChecked((prev) => {
      const next = new Set(prev)
      if (next.has(name)) {
        next.delete(name)
      } else {
        next.add(name)
      }
      return next
    })
  }

  const handleSave = () => {
    // Preserve order based on property displayOrder
    const ordered = availableProps
      .filter((p) => checked.has(p.name))
      .map((p) => p.name)
    onSave(ordered)
    onOpenChange(false)
  }

  const handleReset = () => {
    onReset()
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Edit Columns</DialogTitle>
          <DialogDescription>Select which properties to show as columns.</DialogDescription>
        </DialogHeader>
        <ScrollArea className="max-h-[400px] pr-4">
          <div className="space-y-3">
            {availableProps.map((prop) => (
              <div key={prop.name} className="flex items-center gap-2">
                <Checkbox
                  id={`col-${prop.name}`}
                  checked={checked.has(prop.name)}
                  onCheckedChange={() => handleToggle(prop.name)}
                  data-testid={`column-checkbox-${prop.name}`}
                />
                <Label htmlFor={`col-${prop.name}`} className="text-sm font-normal cursor-pointer">
                  {prop.label}
                </Label>
              </div>
            ))}
          </div>
        </ScrollArea>
        <div className="flex justify-between pt-2">
          <Button variant="outline" size="sm" onClick={handleReset} data-testid="reset-columns-btn">
            Reset to Default
          </Button>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button size="sm" onClick={handleSave} disabled={checked.size === 0} data-testid="save-columns-btn">
              Save
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
