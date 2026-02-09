import { Button } from '@/components/ui/button';
import { ChevronLeft, ChevronRight } from 'lucide-react';

interface PaginationProps {
  hasNext: boolean;
  hasPrevious: boolean;
  onNext: () => void;
  onPrevious: () => void;
  pageLabel?: string;
}

export function Pagination({
  hasNext,
  hasPrevious,
  onNext,
  onPrevious,
  pageLabel,
}: PaginationProps) {
  return (
    <div className="flex items-center justify-between px-2 py-4">
      <span className="text-sm text-muted-foreground">{pageLabel ?? ''}</span>
      <div className="flex gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={onPrevious}
          disabled={!hasPrevious}
        >
          <ChevronLeft className="mr-1 h-4 w-4" />
          Previous
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={onNext}
          disabled={!hasNext}
        >
          Next
          <ChevronRight className="ml-1 h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}
