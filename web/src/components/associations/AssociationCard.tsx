import { useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ChevronDown, ChevronRight, Plus, X } from 'lucide-react'

interface AssociatedObject {
  id: string
  displayValue: string
}

interface AssociationCardProps {
  objectType: string
  items: AssociatedObject[]
  onAdd?: () => void
  onRemove?: (id: string) => void
}

export function AssociationCard({ objectType, items, onAdd, onRemove }: AssociationCardProps) {
  const [expanded, setExpanded] = useState(true)
  const navigate = useNavigate()

  const label = objectType.charAt(0).toUpperCase() + objectType.slice(1)

  return (
    <Card className="gap-0 py-0">
      <CardHeader className="px-4 py-3">
        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="sm"
            className="h-auto flex-1 justify-start gap-2 p-0 font-semibold"
            onClick={() => setExpanded(!expanded)}
          >
            {expanded ? (
              <ChevronDown className="h-4 w-4" />
            ) : (
              <ChevronRight className="h-4 w-4" />
            )}
            <CardTitle className="text-sm">{label}</CardTitle>
            <Badge variant="secondary" className="ml-auto">
              {items.length}
            </Badge>
          </Button>
          {onAdd && (
            <Button
              variant="ghost"
              size="sm"
              className="h-6 w-6 p-0"
              data-testid={`association-add-${objectType}`}
              onClick={(e) => {
                e.stopPropagation()
                onAdd()
              }}
            >
              <Plus className="h-3.5 w-3.5" />
            </Button>
          )}
        </div>
      </CardHeader>
      {expanded && items.length > 0 && (
        <CardContent className="px-4 pb-3 pt-0">
          <ul className="space-y-1">
            {items.map((item) => (
              <li key={item.id}>
                <div className="group flex items-center rounded-md hover:bg-accent transition-colors">
                  <button
                    className="flex-1 px-2 py-1.5 text-left text-sm"
                    onClick={() => {
                      if (item.id !== '__more') {
                        void navigate({
                          to: '/$objectType/$objectId',
                          params: { objectType, objectId: item.id },
                        })
                      }
                    }}
                  >
                    <span className="truncate">{item.displayValue}</span>
                  </button>
                  {onRemove && item.id !== '__more' && (
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100 transition-opacity"
                      data-testid={`association-remove-${item.id}`}
                      onClick={(e) => {
                        e.stopPropagation()
                        onRemove(item.id)
                      }}
                    >
                      <X className="h-3.5 w-3.5" />
                    </Button>
                  )}
                </div>
              </li>
            ))}
          </ul>
        </CardContent>
      )}
    </Card>
  )
}
