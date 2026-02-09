import { useState } from 'react'
import { toast } from 'sonner'
import { RotateCcw, ShieldAlert } from 'lucide-react'
import { useResetData } from '@/api/hooks/useAdmin'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Separator } from '@/components/ui/separator'
import { RequestLog } from '@/components/views/RequestLog'

export function AdminControls() {
  const [confirmReset, setConfirmReset] = useState(false)
  const resetMutation = useResetData()

  const handleReset = () => {
    resetMutation.mutate(undefined, {
      onSuccess: () => {
        toast.success('Data has been reset and re-seeded')
        setConfirmReset(false)
      },
      onError: (err) => toast.error(`Failed to reset data: ${err.message}`),
    })
  }

  return (
    <div className="space-y-6 p-6">
      <div>
        <h1 className="text-2xl font-bold">Admin</h1>
        <p className="text-sm text-muted-foreground">
          Server administration and debugging tools
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <ShieldAlert className="h-5 w-5" />
            Data Controls
          </CardTitle>
          <CardDescription>
            Manage the mock server&apos;s data state
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">Reset All Data</p>
              <p className="text-sm text-muted-foreground">
                Delete all data and re-seed with defaults
              </p>
            </div>
            <Button variant="destructive" onClick={() => setConfirmReset(true)}>
              <RotateCcw className="mr-2 h-4 w-4" />
              Reset Data
            </Button>
          </div>
        </CardContent>
      </Card>

      <Separator />

      <RequestLog />

      {/* Reset Confirmation Dialog */}
      <Dialog open={confirmReset} onOpenChange={setConfirmReset}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Reset All Data</DialogTitle>
            <DialogDescription>
              This will delete all data and re-seed defaults. All objects,
              properties, pipelines, and associations will be reset to their
              initial state. This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirmReset(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleReset}
              disabled={resetMutation.isPending}
            >
              <RotateCcw className="mr-2 h-4 w-4" />
              Reset Everything
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
