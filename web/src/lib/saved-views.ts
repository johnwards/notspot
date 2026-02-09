import type { FilterGroup, Sort } from '@/api/types'

export interface SavedView {
  id: string
  name: string
  objectType: string
  filterGroups: FilterGroup[]
  sorts: Sort[]
  createdAt: string
}

function storageKey(objectType: string): string {
  return `notspot_views_${objectType}`
}

function generateId(): string {
  return `view_${Date.now()}_${Math.random().toString(36).slice(2, 9)}`
}

export function getViews(objectType: string): SavedView[] {
  try {
    const raw = localStorage.getItem(storageKey(objectType))
    if (!raw) return []
    return JSON.parse(raw) as SavedView[]
  } catch {
    return []
  }
}

export function saveView(
  view: Omit<SavedView, 'id' | 'createdAt'>,
): SavedView {
  const views = getViews(view.objectType)
  const newView: SavedView = {
    ...view,
    id: generateId(),
    createdAt: new Date().toISOString(),
  }
  views.push(newView)
  localStorage.setItem(storageKey(view.objectType), JSON.stringify(views))
  return newView
}

export function deleteView(objectType: string, viewId: string): void {
  const views = getViews(objectType).filter((v) => v.id !== viewId)
  localStorage.setItem(storageKey(objectType), JSON.stringify(views))
}

export function renameView(
  objectType: string,
  viewId: string,
  name: string,
): void {
  const views = getViews(objectType).map((v) =>
    v.id === viewId ? { ...v, name } : v,
  )
  localStorage.setItem(storageKey(objectType), JSON.stringify(views))
}
