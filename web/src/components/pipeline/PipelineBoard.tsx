import { useState, useMemo, useCallback } from 'react';
import {
  DndContext,
  DragOverlay,
  type DragStartEvent,
  type DragEndEvent,
  closestCorners,
  PointerSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { usePipelines } from '@/api/hooks/usePipelines';
import { useObjects, useUpdateObject } from '@/api/hooks/useObjects';
import { PipelineColumn } from './PipelineColumn';
import { PipelineCard } from './PipelineCard';
import type { CrmObject, Pipeline, PipelineStage } from '@/api/types';

/** Map object type to the property that stores the pipeline stage */
function getStageProperty(objectType: string): string {
  if (objectType === 'tickets') return 'hs_pipeline_stage';
  return 'dealstage'; // deals and default
}

/** Properties needed for board display and grouping */
function getBoardProperties(objectType: string): string[] {
  if (objectType === 'tickets') {
    return ['subject', 'hs_pipeline_stage', 'hs_ticket_priority', 'createdate'];
  }
  return ['dealname', 'dealstage', 'amount', 'closedate'];
}

function BoardSkeleton() {
  return (
    <div className="flex gap-4 overflow-x-auto p-4">
      {Array.from({ length: 4 }).map((_, i) => (
        <div key={i} className="min-w-[280px] space-y-3">
          <Skeleton className="h-10 w-full rounded-lg" />
          <Skeleton className="h-24 w-full rounded-lg" />
          <Skeleton className="h-24 w-full rounded-lg" />
        </div>
      ))}
    </div>
  );
}

interface PipelineBoardProps {
  objectType: string;
}

export function PipelineBoard({ objectType }: PipelineBoardProps) {
  const stageProperty = getStageProperty(objectType);

  const { data: pipelineData, isLoading: pipelinesLoading } = usePipelines(objectType);
  const { data: objectData, isLoading: objectsLoading } = useObjects(objectType, {
    limit: 100,
    properties: getBoardProperties(objectType),
  });
  const updateObject = useUpdateObject(objectType);

  const pipelines = pipelineData?.results ?? [];
  const [selectedPipelineId, setSelectedPipelineId] = useState<string | null>(null);

  // Local optimistic state for objects during drag
  const [localObjects, setLocalObjects] = useState<CrmObject[] | null>(null);
  // Track the actively dragged card for DragOverlay
  const [activeId, setActiveId] = useState<string | null>(null);

  const currentPipeline: Pipeline | undefined = useMemo(() => {
    if (pipelines.length === 0) return undefined;
    if (selectedPipelineId) {
      return pipelines.find((p) => p.id === selectedPipelineId) ?? pipelines[0];
    }
    return pipelines[0];
  }, [pipelines, selectedPipelineId]);

  const stages: PipelineStage[] = useMemo(() => {
    if (!currentPipeline) return [];
    return [...currentPipeline.stages].sort(
      (a, b) => a.displayOrder - b.displayOrder,
    );
  }, [currentPipeline]);

  // Use local objects for optimistic updates, fall back to server data
  const objects = localObjects ?? objectData?.results ?? [];

  // Group objects by stage
  const objectsByStage = useMemo(() => {
    const map = new Map<string, CrmObject[]>();
    for (const stage of stages) {
      map.set(stage.id, []);
    }
    for (const obj of objects) {
      const stageId = obj.properties[stageProperty] ?? '';
      const arr = map.get(stageId);
      if (arr) {
        arr.push(obj);
      }
      // Objects in stages that don't belong to this pipeline are skipped
    }
    return map;
  }, [objects, stages, stageProperty]);

  // Drag-and-drop sensors — require a small distance to distinguish click from drag
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
  );

  const handleDragStart = useCallback((event: DragStartEvent) => {
    setActiveId(String(event.active.id));
  }, []);

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      setActiveId(null);
      const { active, over } = event;
      if (!over) return;

      const objectId = String(active.id);
      // The "over" target could be a card or a column (droppable).
      // Column droppable IDs match stage IDs.
      let targetStageId = String(over.id);

      // If we dropped over another card, find its stage
      const targetObject = objects.find((o) => o.id === targetStageId);
      if (targetObject) {
        targetStageId = targetObject.properties[stageProperty] ?? '';
      }

      const draggedObject = objects.find((o) => o.id === objectId);
      if (!draggedObject) return;

      const currentStage = draggedObject.properties[stageProperty];
      if (currentStage === targetStageId) return;

      // Optimistically update local state
      const updated = objects.map((o) =>
        o.id === objectId
          ? {
              ...o,
              properties: { ...o.properties, [stageProperty]: targetStageId },
            }
          : o,
      );
      setLocalObjects(updated);

      // Persist via PATCH
      updateObject.mutate(
        { id: objectId, properties: { [stageProperty]: targetStageId } },
        {
          onError: () => {
            // Revert on error
            setLocalObjects(null);
          },
          onSuccess: () => {
            // Clear optimistic state so fresh server data takes over
            setLocalObjects(null);
          },
        },
      );
    },
    [objects, stageProperty, updateObject],
  );

  const isLoading = pipelinesLoading || objectsLoading;

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="flex items-center gap-3 px-4 pt-4">
          <Skeleton className="h-9 w-48" />
        </div>
        <BoardSkeleton />
      </div>
    );
  }

  if (!currentPipeline) {
    return (
      <div className="flex items-center justify-center h-64">
        <p className="text-muted-foreground">No pipelines found for {objectType}.</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Header with pipeline selector */}
      <div className="flex items-center gap-3 px-4 pt-4">
        {pipelines.length > 1 ? (
          <Select
            value={currentPipeline.id}
            onValueChange={setSelectedPipelineId}
          >
            <SelectTrigger className="w-[240px]">
              <SelectValue placeholder="Select pipeline" />
            </SelectTrigger>
            <SelectContent>
              {pipelines.map((p) => (
                <SelectItem key={p.id} value={p.id}>
                  {p.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        ) : (
          <h2 className="text-lg font-semibold">{currentPipeline.label}</h2>
        )}
        <span className="text-sm text-muted-foreground">
          {objects.length} {objects.length === 1 ? 'item' : 'items'}
        </span>
      </div>

      {/* Kanban board */}
      <DndContext
        sensors={sensors}
        collisionDetection={closestCorners}
        onDragStart={handleDragStart}
        onDragEnd={handleDragEnd}
      >
        <div className="flex gap-4 overflow-x-auto px-4 pb-4">
          {stages.map((stage) => (
            <PipelineColumn
              key={stage.id}
              stage={stage}
              objects={objectsByStage.get(stage.id) ?? []}
              objectType={objectType}
            />
          ))}
        </div>
        <DragOverlay>
          {activeId ? (
            <PipelineCard
              object={objects.find((o) => o.id === activeId)!}
              objectType={objectType}
            />
          ) : null}
        </DragOverlay>
      </DndContext>
    </div>
  );
}
