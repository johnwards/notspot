import { useMatches } from '@tanstack/react-router'
import { Search } from 'lucide-react'
import { Button } from '@/components/ui/button'

interface HeaderProps {
  onSearchClick?: () => void
}

export function Header({ onSearchClick }: HeaderProps) {
  const matches = useMatches()

  // Build breadcrumbs from route matches
  const crumbs: string[] = []
  for (const match of matches) {
    const params = match.params as Record<string, string>
    if (params.objectType) {
      crumbs.push(params.objectType.charAt(0).toUpperCase() + params.objectType.slice(1))
    }
    if (params.objectId) {
      crumbs.push(`#${params.objectId}`)
    }
    if (match.pathname.includes('/settings/properties')) {
      crumbs.push('Settings', 'Properties')
    } else if (match.pathname.includes('/settings/pipelines')) {
      crumbs.push('Settings', 'Pipelines')
    } else if (match.pathname.endsWith('/admin')) {
      crumbs.push('Admin')
    }
  }

  // Deduplicate
  const uniqueCrumbs = [...new Set(crumbs)]

  return (
    <header className="flex h-14 items-center justify-between border-b bg-background px-6">
      <nav className="flex items-center gap-1 text-sm text-muted-foreground">
        <span className="font-medium text-foreground">Notspot</span>
        {uniqueCrumbs.map((crumb) => (
          <span key={crumb} className="flex items-center gap-1">
            <span>/</span>
            <span className="font-medium text-foreground">{crumb}</span>
          </span>
        ))}
      </nav>

      <Button variant="outline" size="sm" className="gap-2 text-muted-foreground" onClick={onSearchClick}>
        <Search className="h-4 w-4" />
        <span className="hidden sm:inline">Search</span>
        <kbd className="pointer-events-none hidden h-5 select-none items-center gap-1 rounded border bg-muted px-1.5 font-mono text-[10px] font-medium text-muted-foreground sm:inline-flex">
          <span className="text-xs">⌘</span>K
        </kbd>
      </Button>
    </header>
  )
}
