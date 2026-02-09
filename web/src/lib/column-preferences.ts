function storageKey(objectType: string): string {
  return `notspot_columns_${objectType}`
}

export function getColumnPreferences(objectType: string): string[] | null {
  try {
    const raw = localStorage.getItem(storageKey(objectType))
    if (!raw) return null
    return JSON.parse(raw) as string[]
  } catch {
    return null
  }
}

export function saveColumnPreferences(objectType: string, columns: string[]): void {
  localStorage.setItem(storageKey(objectType), JSON.stringify(columns))
}

export function clearColumnPreferences(objectType: string): void {
  localStorage.removeItem(storageKey(objectType))
}
