import { RecentRecords } from '@/components/dashboard/RecentRecords'
import { DealPipelineSummary } from '@/components/dashboard/DealPipelineSummary'
import { TasksSummary } from '@/components/dashboard/TasksSummary'

export function DashboardPage() {
  return (
    <div className="flex-1 space-y-6 p-6">
      <h1 className="text-2xl font-bold">Dashboard</h1>
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        <RecentRecords />
        <DealPipelineSummary />
        <TasksSummary />
      </div>
    </div>
  )
}
