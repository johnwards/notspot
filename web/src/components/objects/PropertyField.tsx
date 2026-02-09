import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { Property } from '@/api/types'

interface PropertyFieldProps {
  property: Property
  value: string
  onChange: (name: string, value: string) => void
  readOnly?: boolean
}

export function PropertyField({ property, value, onChange, readOnly }: PropertyFieldProps) {
  const isReadOnly = readOnly || property.calculated

  const handleChange = (val: string) => {
    if (!isReadOnly) {
      onChange(property.name, val)
    }
  }

  const renderField = () => {
    switch (property.fieldType) {
      case 'textarea':
        return (
          <textarea
            id={property.name}
            value={value}
            onChange={(e) => handleChange(e.target.value)}
            disabled={isReadOnly}
            className="border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring flex min-h-[80px] w-full rounded-md border px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
            rows={3}
          />
        )

      case 'number':
        return (
          <Input
            id={property.name}
            type="number"
            value={value}
            onChange={(e) => handleChange(e.target.value)}
            disabled={isReadOnly}
          />
        )

      case 'date':
        return (
          <Input
            id={property.name}
            type="date"
            value={value}
            onChange={(e) => handleChange(e.target.value)}
            disabled={isReadOnly}
          />
        )

      case 'select':
      case 'radio':
        return (
          <Select
            value={value}
            onValueChange={handleChange}
            disabled={isReadOnly}
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Select..." />
            </SelectTrigger>
            <SelectContent>
              {property.options
                ?.filter((o) => !o.hidden)
                .sort((a, b) => a.displayOrder - b.displayOrder)
                .map((opt) => (
                  <SelectItem key={opt.value} value={opt.value}>
                    {opt.label}
                  </SelectItem>
                ))}
            </SelectContent>
          </Select>
        )

      case 'checkbox':
      case 'booleancheckbox':
        return (
          <div className="flex items-center gap-2">
            <Checkbox
              id={property.name}
              checked={value === 'true'}
              onCheckedChange={(checked) => handleChange(checked ? 'true' : 'false')}
              disabled={isReadOnly}
            />
            <Label htmlFor={property.name} className="text-sm font-normal">
              {property.label}
            </Label>
          </div>
        )

      case 'phonenumber':
        return (
          <Input
            id={property.name}
            type="tel"
            value={value}
            onChange={(e) => handleChange(e.target.value)}
            disabled={isReadOnly}
          />
        )

      default:
        return (
          <Input
            id={property.name}
            type="text"
            value={value}
            onChange={(e) => handleChange(e.target.value)}
            disabled={isReadOnly}
          />
        )
    }
  }

  // For checkbox types, label is inline
  if (property.fieldType === 'checkbox' || property.fieldType === 'booleancheckbox') {
    return <div className="space-y-1">{renderField()}</div>
  }

  return (
    <div className="space-y-1.5">
      <Label htmlFor={property.name}>
        {property.label}
        {isReadOnly && (
          <span className="ml-1 text-xs text-muted-foreground">(read-only)</span>
        )}
      </Label>
      {renderField()}
      {property.description && (
        <p className="text-xs text-muted-foreground">{property.description}</p>
      )}
    </div>
  )
}
