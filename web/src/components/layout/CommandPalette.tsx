import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { useNavigate } from '@tanstack/react-router'
import {
  CommandDialog,
  CommandInput,
  CommandList,
  CommandEmpty,
  CommandGroup,
  CommandItem,
} from '@/components/ui/command'
import { useSchemas } from '@/api/hooks/useSchemas'
import { apiFetch } from '@/api/client'
import type { SearchResult } from '@/api/types'
import { Loader2 } from 'lucide-react'

const SEARCH_TYPES = ['contacts', 'companies', 'deals', 'tickets'] as const

const DISPLAY_PROPERTY_FALLBACKS: Record<string, string> = {
  contacts: 'email',
  companies: 'name',
  deals: 'dealname',
  tickets: 'subject',
}

interface SearchResultItem {
  objectType: string
  id: string
  displayValue: string
}

interface CommandPaletteProps {
  open?: boolean
  onOpenChange?: (open: boolean) => void
}

export function CommandPalette({ open: controlledOpen, onOpenChange }: CommandPaletteProps) {
  const [internalOpen, setInternalOpen] = useState(false)
  const open = controlledOpen ?? internalOpen
  const setOpen = onOpenChange ?? setInternalOpen

  const [query, setQuery] = useState('')
  const [results, setResults] = useState<SearchResultItem[]>([])
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()
  const { data: schemasData } = useSchemas()
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(null)
  const abortRef = useRef<AbortController>(null)

  // Build a map of objectType -> primaryDisplayProperty
  const displayPropertyMap = useMemo(() => {
    const map: Record<string, string> = { ...DISPLAY_PROPERTY_FALLBACKS }
    if (schemasData?.results) {
      for (const schema of schemasData.results) {
        if (schema.primaryDisplayProperty) {
          map[schema.name] = schema.primaryDisplayProperty
        }
      }
    }
    return map
  }, [schemasData])

  // Keyboard shortcut to open
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setOpen(!open)
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [open, setOpen])

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

      const searches = SEARCH_TYPES.map(async (objectType) => {
        try {
          const data = await apiFetch<SearchResult>(
            `/crm/v3/objects/${objectType}/search`,
            {
              method: 'POST',
              body: JSON.stringify({ query: query.trim(), limit: 5 }),
              signal: controller.signal,
            },
          )
          const displayProp = displayPropertyMap[objectType] ?? 'name'
          return data.results.map((obj) => ({
            objectType,
            id: obj.id,
            displayValue: obj.properties[displayProp] || `${objectType} #${obj.id}`,
          }))
        } catch {
          return []
        }
      })

      Promise.all(searches).then((groups) => {
        if (!controller.signal.aborted) {
          setResults(groups.flat())
          setLoading(false)
        }
      })
    }, 300)

    return () => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current)
      }
    }
  }, [query, displayPropertyMap])

  // Reset on close
  useEffect(() => {
    if (!open) {
      setQuery('')
      setResults([])
      setLoading(false)
    }
  }, [open])

  const handleSelect = useCallback(
    (item: SearchResultItem) => {
      setOpen(false)
      void navigate({
        to: '/$objectType/$objectId',
        params: { objectType: item.objectType, objectId: item.id },
      })
    },
    [setOpen, navigate],
  )

  // Group results by object type
  const grouped = useMemo(() => {
    const groups: Record<string, SearchResultItem[]> = {}
    for (const item of results) {
      if (!groups[item.objectType]) {
        groups[item.objectType] = []
      }
      groups[item.objectType].push(item)
    }
    return groups
  }, [results])

  return (
    <CommandDialog
      open={open}
      onOpenChange={setOpen}
      title="Search"
      description="Search across all CRM objects"
      showCloseButton={false}
    >
      <CommandInput
        placeholder="Search contacts, companies, deals, tickets..."
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
        {!loading &&
          Object.entries(grouped).map(([objectType, items]) => (
            <CommandGroup
              key={objectType}
              heading={objectType.charAt(0).toUpperCase() + objectType.slice(1)}
            >
              {items.map((item) => (
                <CommandItem
                  key={`${item.objectType}-${item.id}`}
                  value={`${item.objectType}-${item.id}-${item.displayValue}`}
                  onSelect={() => handleSelect(item)}
                >
                  <span className="truncate">{item.displayValue}</span>
                  <span className="ml-auto text-xs text-muted-foreground">#{item.id}</span>
                </CommandItem>
              ))}
            </CommandGroup>
          ))}
      </CommandList>
    </CommandDialog>
  )
}
