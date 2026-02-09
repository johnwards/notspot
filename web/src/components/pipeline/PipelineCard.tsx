import { useSortable } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { cn } from '@/lib/utils';
import type { CrmObject } from '@/api/types';

interface PipelineCardProps {
  object: CrmObject;
  objectType: string;
  onClick?: (object: CrmObject) => void;
}

function formatCurrency(value: string | undefined): string {
  if (!value) return '';
  const num = parseFloat(value);
  if (isNaN(num)) return value;
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(num);
}

function formatDate(value: string | undefined): string {
  if (!value) return '';
  try {
    return new Date(value).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    });
  } catch {
    return value;
  }
}

function DealDetails({ properties }: { properties: Record<string, string> }) {
  return (
    <>
      {properties.amount && (
        <p className="text-sm font-semibold text-foreground">
          {formatCurrency(properties.amount)}
        </p>
      )}
      {properties.closedate && (
        <p className="text-xs text-muted-foreground">
          Close: {formatDate(properties.closedate)}
        </p>
      )}
    </>
  );
}

function TicketDetails({ properties }: { properties: Record<string, string> }) {
  return (
    <>
      {properties.hs_ticket_priority && (
        <p className="text-xs text-muted-foreground">
          Priority: {properties.hs_ticket_priority}
        </p>
      )}
      {properties.createdate && (
        <p className="text-xs text-muted-foreground">
          Created: {formatDate(properties.createdate)}
        </p>
      )}
    </>
  );
}

function getDisplayName(object: CrmObject, objectType: string): string {
  const { properties } = object;
  if (objectType === 'deals') {
    return properties.dealname || properties.hs_object_id || object.id;
  }
  if (objectType === 'tickets') {
    return properties.subject || properties.hs_object_id || object.id;
  }
  // Fallback: try common display properties
  return (
    properties.name ||
    properties.dealname ||
    properties.subject ||
    properties.hs_object_id ||
    object.id
  );
}

export function PipelineCard({ object, objectType, onClick }: PipelineCardProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: object.id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...listeners}
      onClick={onClick ? () => onClick(object) : undefined}
      className={cn(
        'rounded-lg border bg-card p-3 shadow-sm cursor-grab active:cursor-grabbing',
        'hover:shadow-md transition-shadow',
        isDragging && 'opacity-50 shadow-lg',
        onClick && 'cursor-pointer',
      )}
    >
      <p className="text-sm font-medium truncate">
        {getDisplayName(object, objectType)}
      </p>
      <div className="mt-1 space-y-0.5">
        {objectType === 'deals' && <DealDetails properties={object.properties} />}
        {objectType === 'tickets' && <TicketDetails properties={object.properties} />}
      </div>
    </div>
  );
}
