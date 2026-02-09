import { useState, useEffect, useCallback, useRef } from 'react'
import {
  CommandDialog,
  CommandInput,
  CommandList,
  CommandEmpty,
  CommandGroup,
  CommandItem,
} from '@/components/ui/command'
import { Button } from '@/components/ui/button'
import { apiFetch } from '@/api/client'
import { useCreateAssociation } from '@/api/hooks/useAssociationMutations'
import { ObjectCreateDialog } from '@/components/objects/ObjectCreateDialog'
import type { SearchResult, CrmObject } from '@/api/types'
import { Loader2, Plus } from 'lucide-react'
import { toast } from 'sonner'
import { singularize } from '@/lib/utils'

const DISPLAY_PROPERTY_FALLBACKS: Record<string, string> = {
  contacts: 'email',
  companies: 'name',
  deals: 'dealname',
  tickets: 'subject',
}

interface AssociationSearchDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  fromType: string
  fromId: string
  toType: string
  onAssociated?: () => void
  /** When set, dialog operates in "pick only" mode — returns the selected record instead of creating an association */
  onSelect?: (record: { id: string; displayValue: string }) => void
}

export function AssociationSearchDialog({
  open,
  onOpenChange,
  fromType,
  fromId,
  toType,
  onAssociated,
  onSelect,
}: AssociationSearchDialogProps) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<CrmObject[]>([])
  const [loading, setLoading] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(null)
  const abortRef = useRef<AbortController>(null)

  const createAssociation = useCreateAssociation(fromType, fromId, toType)
  const displayProp = DISPLAY_PROPERTY_FALLBACKS[toType] ?? 'name'

  // Search when query changes (debounced)
  useEffect(() => {
    if (debounceRef.current) {
      clearTimeout(debounceRef.current)
    }
    if (abortRef.current) {
      abortRef.current.abort()
    }

    if (!query.trim()) {
      setResults([])
      setLoading(false)
      return
    }

    setLoading(true)
    debounceRef.current = setTimeout(() => {
      const controller = new AbortController()
      abortRef.current = controller

      apiFetch<SearchResult>(`/crm/v3/objects/${toType}/search`, {
        method: 'POST',
        body: JSON.stringify({ query: query.trim(), limit: 10 }),
        signal: controller.signal,
      })
        .then((data) => {
          if (!controller.signal.aborted) {
            setResults(data.results)
            setLoading(false)
          }
        })
        .catch(() => {
          if (!controller.signal.aborted) {
            setResults([])
            setLoading(false)
          }
        })
    }, 300)

    return () => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current)
      }
    }
  }, [query, toType])

  // Reset on close
  useEffect(() => {
    if (!open) {
      setQuery('')
      setResults([])
      setLoading(false)
    }
  }, [open])

  const handleSelect = useCallback(
    (obj: CrmObject) => {
      if (onSelect) {
        // "Pick only" mode — return selection without creating association
        const displayValue = obj.properties[displayProp] || `${toType} #${obj.id}`
        onSelect({ id: obj.id, displayValue })
        onOpenChange(false)
        return
      }
      createAssociation.mutate(obj.id, {
        onSuccess: () => {
          toast.success('Association created')
          onOpenChange(false)
          onAssociated?.()
        },
        onError: (err) => {
          toast.error(`Failed to associate: ${err.message}`)
        },
      })
    },
    [createAssociation, onOpenChange, onAssociated, onSelect, displayProp, toType],
  )

  const handleCreated = useCallback(() => {
    // After creating a new object via the create dialog, close it and
    // let the user search again to find the newly created record
    setCreateOpen(false)
    toast.success(`${singularize(toType)} created — search to associate it`)
  }, [toType])

  const label = toType.charAt(0).toUpperCase() + toType.slice(1)

  return (
    <>
      <CommandDialog
        open={open && !createOpen}
        onOpenChange={onOpenChange}
        title={`Associate ${label}`}
        description={`Search for ${toType} to associate`}
        showCloseButton={false}
        shouldFilter={false}
      >
        <CommandInput
          placeholder={`Search ${toType}...`}
          value={query}
          onValueChange={setQuery}
        />
        <CommandList>
          {loading && (
            <div className="flex items-center justify-center py-6">
              <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
            </div>
          )}
          {!loading && query.trim() && results.length === 0 && (
            <CommandEmpty>No results found.</CommandEmpty>
          )}
          {!loading && results.length > 0 && (
            <CommandGroup heading={label}>
              {results.map((obj) => (
                <CommandItem
                  key={obj.id}
                  value={`${obj.id}-${obj.properties[displayProp] ?? ''}`}
                  onSelect={() => handleSelect(obj)}
                  disabled={createAssociation.isPending}
                >
                  <span className="truncate">
                    {obj.properties[displayProp] || `${toType} #${obj.id}`}
                  </span>
                  <span className="ml-auto text-xs text-muted-foreground">
                    #{obj.id}
                  </span>
                </CommandItem>
              ))}
            </CommandGroup>
          )}
        </CommandList>
        <div className="border-t p-2">
          <Button
            variant="ghost"
            size="sm"
            className="w-full justify-start gap-2"
            onClick={() => setCreateOpen(true)}
          >
            <Plus className="h-4 w-4" />
            Create new {singularize(toType)}
          </Button>
        </div>
      </CommandDialog>

      <ObjectCreateDialog
        objectType={toType}
        open={createOpen}
        onOpenChange={(isOpen) => {
          setCreateOpen(isOpen)
          if (!isOpen) {
            handleCreated()
          }
        }}
      />
    </>
  )
}
