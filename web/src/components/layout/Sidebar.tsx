import { useState } from 'react'
import { Link, useMatchRoute } from '@tanstack/react-router'
import { useSchemas } from '@/api/hooks/useSchemas'
import {
  Users,
  Building2,
  Handshake,
  Ticket,
  List,
  GitBranch,
  Settings,
  Kanban,
  LayoutDashboard,
  StickyNote,
  Phone,
  Mail,
  CheckSquare,
  Calendar,
  ChevronDown,
  ChevronRight,
} from 'lucide-react'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type { LucideIcon } from 'lucide-react'

const engagementTypes: { name: string; label: string; icon: LucideIcon }[] = [
  { name: 'notes', label: 'Notes', icon: StickyNote },
  { name: 'calls', label: 'Calls', icon: Phone },
  { name: 'emails', label: 'Emails', icon: Mail },
  { name: 'tasks', label: 'Tasks', icon: CheckSquare },
  { name: 'meetings', label: 'Meetings', icon: Calendar },
]

const defaultObjectTypes: { name: string; label: string; icon: LucideIcon; hasBoard?: boolean }[] = [
  { name: 'contacts', label: 'Contacts', icon: Users },
  { name: 'companies', label: 'Companies', icon: Building2 },
  { name: 'deals', label: 'Deals', icon: Handshake, hasBoard: true },
  { name: 'tickets', label: 'Tickets', icon: Ticket, hasBoard: true },
]

interface NavItemProps {
  to: string
  icon: LucideIcon
  label: string
}

function NavItem({ to, icon: Icon, label }: NavItemProps) {
  const matchRoute = useMatchRoute()
  const isActive = !!matchRoute({ to, fuzzy: true })

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Link
          to={to}
          className={`flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-all duration-200 ${
            isActive
              ? 'bg-sidebar-accent text-sidebar-primary'
              : 'text-sidebar-foreground hover:bg-sidebar-accent/50 hover:translate-x-0.5'
          }`}
        >
          <Icon className="h-4 w-4 shrink-0" />
          <span className="hidden lg:inline">{label}</span>
        </Link>
      </TooltipTrigger>
      <TooltipContent side="right" className="lg:hidden">
        {label}
      </TooltipContent>
    </Tooltip>
  )
}

export function Sidebar() {
  const [engagementsOpen, setEngagementsOpen] = useState(true)
  const { data: schemasData } = useSchemas()

  const pipelineTypes = new Set(['deals', 'tickets'])

  // Use schema data for CRM types if available, else fall back to defaults
  const crmTypes = schemasData?.results?.length
    ? schemasData.results.map((s) => {
        const def = defaultObjectTypes.find((d) => d.name === s.name)
        return {
          name: s.name,
          label: s.labels?.plural ?? s.name,
          icon: def?.icon ?? List,
          hasBoard: pipelineTypes.has(s.name),
        }
      })
    : defaultObjectTypes

  return (
    <TooltipProvider>
      <div className="flex h-full w-14 lg:w-60 flex-col bg-sidebar text-sidebar-foreground transition-all duration-200">
        <div className="flex h-14 items-center px-4">
          <span className="text-lg font-bold text-white hidden lg:inline">Notspot</span>
          <span className="text-lg font-bold text-white lg:hidden">N</span>
        </div>

        <ScrollArea className="flex-1 px-2 lg:px-3">
          <div className="space-y-6 py-2">
            {/* CRM Section */}
            <div>
              <p className="mb-2 px-3 text-xs font-semibold uppercase tracking-wider text-sidebar-foreground/50 hidden lg:block">
                CRM
              </p>
              <nav className="space-y-1">
                <NavItem to="/dashboard" icon={LayoutDashboard} label="Home" />
                {crmTypes.map((t) => (
                  <div key={t.name}>
                    <NavItem
                      to={`/${t.name}`}
                      icon={t.icon}
                      label={t.label}
                    />
                    {t.hasBoard && (
                      <NavItem
                        to={`/${t.name}/board`}
                        icon={Kanban}
                        label={`${t.label} Board`}
                      />
                    )}
                  </div>
                ))}
              </nav>
            </div>

            {/* Engagements Section */}
            <div data-testid="engagements-section">
              <button
                onClick={() => setEngagementsOpen((v) => !v)}
                className="mb-2 px-3 flex items-center gap-1 text-xs font-semibold uppercase tracking-wider text-sidebar-foreground/50 w-full text-left"
                data-testid="engagements-toggle"
              >
                {engagementsOpen ? (
                  <ChevronDown className="h-3 w-3 shrink-0" />
                ) : (
                  <ChevronRight className="h-3 w-3 shrink-0" />
                )}
                <span className="hidden lg:inline">Engagements</span>
              </button>
              {engagementsOpen && (
                <nav className="space-y-1">
                  {engagementTypes.map((t) => (
                    <NavItem
                      key={t.name}
                      to={`/${t.name}`}
                      icon={t.icon}
                      label={t.label}
                    />
                  ))}
                </nav>
              )}
            </div>

            {/* Settings Section */}
            <div>
              <p className="mb-2 px-3 text-xs font-semibold uppercase tracking-wider text-sidebar-foreground/50 hidden lg:block">
                Settings
              </p>
              <nav className="space-y-1">
                <NavItem to="/settings/properties" icon={List} label="Properties" />
                <NavItem to="/settings/pipelines" icon={GitBranch} label="Pipelines" />
              </nav>
            </div>

            {/* Admin Section */}
            <div>
              <p className="mb-2 px-3 text-xs font-semibold uppercase tracking-wider text-sidebar-foreground/50 hidden lg:block">
                Admin
              </p>
              <nav className="space-y-1">
                <NavItem to="/admin" icon={Settings} label="Admin" />
              </nav>
            </div>
          </div>
        </ScrollArea>
      </div>
    </TooltipProvider>
  )
}
