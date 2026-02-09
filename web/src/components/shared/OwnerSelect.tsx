import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useOwners } from '@/api/hooks/useOwners'

interface OwnerSelectProps {
  value: string
  onValueChange: (value: string) => void
  disabled?: boolean
}

export function OwnerSelect({ value, onValueChange, disabled }: OwnerSelectProps) {
  const { data: ownersData } = useOwners()
  const owners = ownersData?.results ?? []

  return (
    <Select value={value || '__none'} onValueChange={(v) => onValueChange(v === '__none' ? '' : v)} disabled={disabled}>
      <SelectTrigger className="w-full" data-testid="owner-select">
        <SelectValue placeholder="No owner" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="__none">No owner</SelectItem>
        {owners.map((owner) => (
          <SelectItem key={owner.id} value={owner.id}>
            {owner.firstName} {owner.lastName}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

export function useOwnerName(ownerId: string): string {
  const { data: ownersData } = useOwners()
  if (!ownerId) return ''
  const owner = ownersData?.results?.find((o) => o.id === ownerId)
  return owner ? `${owner.firstName} ${owner.lastName}` : ownerId
}
