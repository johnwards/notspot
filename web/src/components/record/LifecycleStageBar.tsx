import { useMemo } from 'react'
import { Card, CardContent } from '@/components/ui/card'
import type { CrmObject } from '@/api/types'

const LIFECYCLE_STAGES = [
  { value: 'subscriber', label: 'Subscriber' },
  { value: 'lead', label: 'Lead' },
  { value: 'marketingqualifiedlead', label: 'MQL' },
  { value: 'salesqualifiedlead', label: 'SQL' },
  { value: 'opportunity', label: 'Opportunity' },
  { value: 'customer', label: 'Customer' },
  { value: 'evangelist', label: 'Evangelist' },
]

interface LifecycleStageBarProps {
  objectType: string
  object: CrmObject
}

export function LifecycleStageBar({ objectType, object }: LifecycleStageBarProps) {
  const isApplicable = objectType === 'contacts' || objectType === 'companies'

  const currentStage = object.properties.lifecyclestage ?? ''

  const currentIndex = useMemo(() => {
    return LIFECYCLE_STAGES.findIndex((s) => s.value === currentStage)
  }, [currentStage])

  if (!isApplicable) return null

  // Don't render if no valid lifecycle stage is set
  if (currentIndex === -1 && !currentStage) return null

  return (
    <Card data-testid="lifecycle-stage-bar">
      <CardContent className="py-3 px-4">
        <p className="text-xs font-medium text-muted-foreground mb-2">Lifecycle Stage</p>
        <div className="flex items-center gap-0.5">
          {LIFECYCLE_STAGES.map((stage, index) => {
            const isFilled = currentIndex >= 0 && index <= currentIndex
            const isCurrent = index === currentIndex
            return (
              <div key={stage.value} className="flex-1 flex flex-col items-center gap-1">
                <div className="relative w-full h-2 rounded-full overflow-hidden">
                  <div
                    className={`h-full rounded-full transition-colors ${
                      isFilled ? 'bg-primary' : 'bg-muted'
                    }`}
                    data-filled={isFilled || undefined}
                    data-stage={stage.value}
                  />
                  {isCurrent && (
                    <div
                      className="absolute top-1/2 right-0 -translate-y-1/2 translate-x-1/2 w-3 h-3 rounded-full bg-primary border-2 border-background z-10"
                      data-testid="lifecycle-current-marker"
                    />
                  )}
                </div>
                <span
                  className={`text-[10px] leading-tight text-center ${
                    isCurrent ? 'font-semibold text-primary' : 'text-muted-foreground'
                  }`}
                  data-testid={`lifecycle-label-${stage.value}`}
                >
                  {stage.label}
                </span>
              </div>
            )
          })}
        </div>
      </CardContent>
    </Card>
  )
}
