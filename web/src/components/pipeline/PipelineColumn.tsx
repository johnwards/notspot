import { useDroppable } from '@dnd-kit/core';
import { SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import { PipelineCard } from './PipelineCard';
import type { CrmObject, PipelineStage } from '@/api/types';

interface PipelineColumnProps {
  stage: PipelineStage;
  objects: CrmObject[];
  objectType: string;
  onCardClick?: (object: CrmObject) => void;
}

function formatCurrency(value: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value);
}

function computeAmountSum(objects: CrmObject[]): number {
  return objects.reduce((sum, obj) => {
    const val = parseFloat(obj.properties.amount || '0');
    return sum + (isNaN(val) ? 0 : val);
  }, 0);
}

function computeWeightedSum(objects: CrmObject[], probability: number): number {
  return objects.reduce((sum, obj) => {
    const val = parseFloat(obj.properties.amount || '0');
    return sum + (isNaN(val) ? 0 : val * probability);
  }, 0);
}

export function PipelineColumn({
  stage,
  objects,
  objectType,
  onCardClick,
}: PipelineColumnProps) {
  const { setNodeRef, isOver } = useDroppable({ id: stage.id });
  const objectIds = objects.map((o) => o.id);
  const showAmount = objectType === 'deals';
  const amountSum = showAmount ? computeAmountSum(objects) : 0;
  const probability = parseFloat(stage.metadata?.probability || '0');
  const weightedSum = showAmount ? computeWeightedSum(objects, isNaN(probability) ? 0 : probability) : 0;

  return (
    <div
      className={cn(
        'flex flex-col rounded-lg border bg-muted/50 min-w-[280px] max-w-[320px]',
        isOver && 'ring-2 ring-primary/40',
      )}
    >
      {/* Column header */}
      <div className="flex items-center justify-between gap-2 px-3 py-2.5 border-b">
        <div className="min-w-0">
          <h3 className="text-sm font-semibold truncate">{stage.label}</h3>
          {showAmount && amountSum > 0 && (
            <p className="text-xs text-muted-foreground" data-testid="column-total">
              Total: {formatCurrency(amountSum)}
            </p>
          )}
          {showAmount && amountSum > 0 && (
            <p className="text-xs text-muted-foreground/70" data-testid="column-weighted">
              Weighted: {formatCurrency(weightedSum)}
            </p>
          )}
        </div>
        <Badge variant="secondary" className="shrink-0">
          {objects.length}
        </Badge>
      </div>

      {/* Droppable card area */}
      <div ref={setNodeRef} className="flex-1 min-h-[100px]">
        <ScrollArea className="h-[calc(100vh-240px)]">
          <SortableContext items={objectIds} strategy={verticalListSortingStrategy}>
            <div className="flex flex-col gap-2 p-2">
              {objects.map((obj) => (
                <PipelineCard
                  key={obj.id}
                  object={obj}
                  objectType={objectType}
                  onClick={onCardClick}
                />
              ))}
              {objects.length === 0 && (
                <p className="text-xs text-muted-foreground text-center py-4">
                  No items
                </p>
              )}
            </div>
          </SortableContext>
        </ScrollArea>
      </div>
    </div>
  );
}
