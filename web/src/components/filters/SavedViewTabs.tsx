import { useState, useCallback, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Bookmark, MoreHorizontal, Pencil, Trash2 } from 'lucide-react'
import {
  getViews,
  saveView,
  deleteView,
  renameView,
  type SavedView,
} from '@/lib/saved-views'
import type { FilterGroup } from '@/api/types'

interface SavedViewTabsProps {
  objectType: string
  activeViewId: string | null
  onViewChange: (viewId: string | null, filterGroups: FilterGroup[]) => void
  currentFilters: FilterGroup[]
}

export function SavedViewTabs({
  objectType,
  activeViewId,
  onViewChange,
  currentFilters,
}: SavedViewTabsProps) {
  const [views, setViews] = useState<SavedView[]>([])
  const [saveOpen, setSaveOpen] = useState(false)
  const [saveName, setSaveName] = useState('')
  const [renamingId, setRenamingId] = useState<string | null>(null)
  const [renameValue, setRenameValue] = useState('')

  // Load views from localStorage
  const refreshViews = useCallback(() => {
    setViews(getViews(objectType))
  }, [objectType])

  useEffect(() => {
    refreshViews()
  }, [refreshViews])

  const handleSave = useCallback(() => {
    if (!saveName.trim()) return
    const view = saveView({
      name: saveName.trim(),
      objectType,
      filterGroups: currentFilters,
      sorts: [],
    })
    setSaveName('')
    setSaveOpen(false)
    refreshViews()
    onViewChange(view.id, view.filterGroups)
  }, [saveName, objectType, currentFilters, refreshViews, onViewChange])

  const handleDelete = useCallback(
    (viewId: string) => {
      deleteView(objectType, viewId)
      refreshViews()
      if (activeViewId === viewId) {
        onViewChange(null, [])
      }
    },
    [objectType, refreshViews, activeViewId, onViewChange],
  )

  const handleRename = useCallback(
    (viewId: string) => {
      if (!renameValue.trim()) return
      renameView(objectType, viewId, renameValue.trim())
      setRenamingId(null)
      setRenameValue('')
      refreshViews()
    },
    [objectType, renameValue, refreshViews],
  )

  const hasFilters = currentFilters.length > 0 && (currentFilters[0]?.filters?.length ?? 0) > 0

  return (
    <div className="flex items-center gap-1 overflow-x-auto pb-1" data-testid="saved-view-tabs">
      {/* All objects tab */}
      <Button
        variant={activeViewId === null ? 'secondary' : 'ghost'}
        size="sm"
        onClick={() => onViewChange(null, [])}
        data-testid="view-tab-all"
      >
        All {objectType}
      </Button>

      {/* Saved view tabs */}
      {views.map((view) => (
        <div key={view.id} className="flex items-center group">
          {renamingId === view.id ? (
            <form
              onSubmit={(e) => {
                e.preventDefault()
                handleRename(view.id)
              }}
              className="flex items-center gap-1"
            >
              <Input
                value={renameValue}
                onChange={(e) => setRenameValue(e.target.value)}
                className="h-8 w-32 text-sm"
                autoFocus
                data-testid="rename-input"
                onBlur={() => {
                  setRenamingId(null)
                  setRenameValue('')
                }}
                onKeyDown={(e) => {
                  if (e.key === 'Escape') {
                    setRenamingId(null)
                    setRenameValue('')
                  }
                }}
              />
            </form>
          ) : (
            <Button
              variant={activeViewId === view.id ? 'secondary' : 'ghost'}
              size="sm"
              onClick={() => onViewChange(view.id, view.filterGroups)}
              data-testid={`view-tab-${view.id}`}
            >
              <Bookmark className="h-3 w-3" />
              {view.name}
            </Button>
          )}

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                size="icon-xs"
                className="opacity-0 group-hover:opacity-100 transition-opacity"
                aria-label={`${view.name} options`}
                data-testid={`view-menu-${view.id}`}
              >
                <MoreHorizontal className="h-3 w-3" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start">
              <DropdownMenuItem
                onClick={() => {
                  setRenamingId(view.id)
                  setRenameValue(view.name)
                }}
                data-testid="rename-view"
              >
                <Pencil className="h-3 w-3" />
                Rename
              </DropdownMenuItem>
              <DropdownMenuItem
                variant="destructive"
                onClick={() => handleDelete(view.id)}
                data-testid="delete-view"
              >
                <Trash2 className="h-3 w-3" />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      ))}

      {/* Save current view button */}
      {hasFilters && (
        <Popover open={saveOpen} onOpenChange={setSaveOpen}>
          <PopoverTrigger asChild>
            <Button
              variant="ghost"
              size="sm"
              className="text-muted-foreground"
              data-testid="save-view-button"
            >
              <Bookmark className="h-3 w-3" />
              Save view
            </Button>
          </PopoverTrigger>
          <PopoverContent className="w-64" align="start">
            <form
              onSubmit={(e) => {
                e.preventDefault()
                handleSave()
              }}
              className="space-y-3"
            >
              <div className="space-y-1">
                <label htmlFor="view-name" className="text-sm font-medium">
                  View name
                </label>
                <Input
                  id="view-name"
                  value={saveName}
                  onChange={(e) => setSaveName(e.target.value)}
                  placeholder="My filter view"
                  autoFocus
                  data-testid="save-view-name"
                />
              </div>
              <Button
                type="submit"
                size="sm"
                className="w-full"
                disabled={!saveName.trim()}
                data-testid="save-view-confirm"
              >
                Save
              </Button>
            </form>
          </PopoverContent>
        </Popover>
      )}
    </div>
  )
}
