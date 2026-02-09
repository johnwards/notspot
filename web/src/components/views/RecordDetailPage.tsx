import { useMemo } from 'react'
import { useObject } from '@/api/hooks/useObjects'
import { useProperties } from '@/api/hooks/useProperties'
import { AboutCard } from '@/components/record/AboutCard'
import { ActionButtons } from '@/components/record/ActionButtons'
import { ActivityTimeline } from '@/components/record/ActivityTimeline'
import { AssociationSidebar } from '@/components/record/AssociationSidebar'
import { LifecycleStageBar } from '@/components/record/LifecycleStageBar'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

interface RecordDetailPageProps {
  objectType: string
  objectId: string
}

export function RecordDetailPage({ objectType, objectId }: RecordDetailPageProps) {
  const { data: propertiesData, isLoading: propsLoading } = useProperties(objectType)
  const properties = propertiesData?.results ?? []
  const propertyNames = useMemo(() => properties.map((p) => p.name), [properties])

  const propsReady = propertyNames.length > 0
  const { data: object, isLoading: objectLoading } = useObject(
    objectType,
    propsReady ? objectId : '',
    propsReady ? propertyNames : undefined,
  )

  const isLoading = propsLoading || objectLoading

  if (isLoading) {
    return (
      <div className="flex gap-6 h-full">
        <div className="w-80 shrink-0 space-y-4">
          <Skeleton className="h-64 w-full" />
        </div>
        <div className="flex-1 space-y-4">
          <Skeleton className="h-10 w-48" />
          <Skeleton className="h-64 w-full" />
        </div>
        <div className="w-80 shrink-0 space-y-4">
          <Skeleton className="h-64 w-full" />
        </div>
      </div>
    )
  }

  if (!object) {
    return <div className="text-muted-foreground">Object not found</div>
  }

  return (
    <div className="flex gap-6 h-full">
      {/* Left Panel */}
      <div className="w-80 shrink-0 space-y-4">
        <LifecycleStageBar objectType={objectType} object={object} />
        <AboutCard
          objectType={objectType}
          objectId={objectId}
          object={object}
          properties={properties}
        />
        <ActionButtons objectType={objectType} objectId={objectId} />
      </div>

      {/* Middle Panel */}
      <div className="flex-1 min-w-0">
        <Tabs defaultValue="overview">
          <TabsList>
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="activity">Activity</TabsTrigger>
          </TabsList>
          <TabsContent value="overview" className="mt-4">
            <div className="space-y-4">
              <h3 className="text-lg font-semibold">Properties</h3>
              <div className="grid grid-cols-2 gap-4">
                {properties
                  .filter((p) => !p.hidden && !p.archived)
                  .sort((a, b) => a.displayOrder - b.displayOrder)
                  .slice(0, 12)
                  .map((prop) => (
                    <div key={prop.name} className="space-y-1">
                      <p className="text-xs text-muted-foreground">{prop.label}</p>
                      <p className="text-sm">{object.properties[prop.name] || '\u2014'}</p>
                    </div>
                  ))}
              </div>
            </div>
          </TabsContent>
          <TabsContent value="activity" className="mt-4">
            <ActivityTimeline objectType={objectType} objectId={objectId} />
          </TabsContent>
        </Tabs>
      </div>

      {/* Right Panel */}
      <div className="w-80 shrink-0">
        <AssociationSidebar objectType={objectType} objectId={objectId} />
      </div>
    </div>
  )
}
