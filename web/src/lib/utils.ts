import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/**
 * Naively singularize an English plural word.
 * Handles: "companies" → "company", "contacts" → "contact", etc.
 */
export function singularize(word: string): string {
  if (word.endsWith('ies')) {
    return word.slice(0, -3) + 'y'
  }
  if (word.endsWith('ses') || word.endsWith('xes') || word.endsWith('zes') || word.endsWith('shes') || word.endsWith('ches')) {
    return word.slice(0, -2)
  }
  if (word.endsWith('s')) {
    return word.slice(0, -1)
  }
  return word
}
