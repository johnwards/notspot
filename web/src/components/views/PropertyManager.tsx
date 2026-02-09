import { useState } from 'react'
import { toast } from 'sonner'
import { Settings2, Plus, Pencil, Trash2 } from 'lucide-react'
import { useProperties, useCreateProperty, useUpdateProperty, useArchiveProperty } from '@/api/hooks/useProperties'
import type { Property } from '@/api/types'
import { DataTable, type Column } from '@/components/shared/DataTable'
import { EmptyState } from '@/components/shared/EmptyState'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

const OBJECT_TYPES = ['contacts', 'companies', 'deals', 'tickets'] as const

const PROPERTY_TYPES = [
  'string',
  'number',
  'date',
  'datetime',
  'enumeration',
  'bool',
  'phone_number',
] as const

const FIELD_TYPES_BY_TYPE: Record<string, string[]> = {
  string: ['text', 'textarea', 'html', 'file'],
  number: ['number'],
  date: ['date'],
  datetime: ['date'],
  enumeration: ['select', 'checkbox', 'radio', 'booleancheckbox'],
  bool: ['booleancheckbox'],
  phone_number: ['phonenumber'],
}

interface PropertyFormData {
  name: string
  label: string
  type: string
  fieldType: string
  groupName: string
  description: string
  options: { label: string; value: string; displayOrder: number; hidden: boolean }[]
}

const emptyForm: PropertyFormData = {
  name: '',
  label: '',
  type: 'string',
  fieldType: 'text',
  groupName: 'contactinformation',
  description: '',
  options: [],
}

export function PropertyManager() {
  const [objectType, setObjectType] = useState<string>('contacts')
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingProperty, setEditingProperty] = useState<string | null>(null)
  const [form, setForm] = useState<PropertyFormData>(emptyForm)
  const [confirmArchive, setConfirmArchive] = useState<string | null>(null)

  const { data, isLoading } = useProperties(objectType)
  const createMutation = useCreateProperty(objectType)
  const updateMutation = useUpdateProperty(objectType)
  const archiveMutation = useArchiveProperty(objectType)

  const properties = data?.results ?? []

  const openCreate = () => {
    setEditingProperty(null)
    setForm(emptyForm)
    setDialogOpen(true)
  }

  const openEdit = (prop: Property) => {
    setEditingProperty(prop.name)
    setForm({
      name: prop.name,
      label: prop.label,
      type: prop.type,
      fieldType: prop.fieldType,
      groupName: prop.groupName,
      description: prop.description,
      options: prop.options ?? [],
    })
    setDialogOpen(true)
  }

  const handleSubmit = () => {
    if (editingProperty) {
      updateMutation.mutate(
        {
          propertyName: editingProperty,
          label: form.label,
          type: form.type,
          fieldType: form.fieldType,
          groupName: form.groupName,
          description: form.description,
          ...(form.type === 'enumeration' ? { options: form.options } : {}),
        },
        {
          onSuccess: () => {
            toast.success('Property updated')
            setDialogOpen(false)
          },
          onError: (err) => toast.error(`Failed to update property: ${err.message}`),
        },
      )
    } else {
      createMutation.mutate(
        {
          name: form.name,
          label: form.label,
          type: form.type,
          fieldType: form.fieldType,
          groupName: form.groupName,
          description: form.description,
          ...(form.type === 'enumeration' ? { options: form.options } : {}),
        },
        {
          onSuccess: () => {
            toast.success('Property created')
            setDialogOpen(false)
          },
          onError: (err) => toast.error(`Failed to create property: ${err.message}`),
        },
      )
    }
  }

  const handleArchive = (name: string) => {
    archiveMutation.mutate(name, {
      onSuccess: () => {
        toast.success('Property archived')
        setConfirmArchive(null)
      },
      onError: (err) => toast.error(`Failed to archive property: ${err.message}`),
    })
  }

  const handleTypeChange = (type: string) => {
    const fieldTypes = FIELD_TYPES_BY_TYPE[type] ?? ['text']
    setForm((f) => ({ ...f, type, fieldType: fieldTypes[0] }))
  }

  const addOption = () => {
    setForm((f) => ({
      ...f,
      options: [...f.options, { label: '', value: '', displayOrder: f.options.length, hidden: false }],
    }))
  }

  const updateOption = (idx: number, key: 'label' | 'value', val: string) => {
    setForm((f) => ({
      ...f,
      options: f.options.map((o, i) => (i === idx ? { ...o, [key]: val } : o)),
    }))
  }

  const removeOption = (idx: number) => {
    setForm((f) => ({ ...f, options: f.options.filter((_, i) => i !== idx) }))
  }

  const columns: Column<Property & Record<string, unknown>>[] = [
    { key: 'name', header: 'Name', sortable: true },
    { key: 'label', header: 'Label', sortable: true },
    {
      key: 'type',
      header: 'Type',
      sortable: true,
      render: (row) => <Badge variant="outline">{row.type}</Badge>,
    },
    { key: 'fieldType', header: 'Field Type', sortable: true },
    { key: 'groupName', header: 'Group', sortable: true },
    {
      key: 'hidden',
      header: 'Hidden',
      render: (row) => (row.hidden ? 'Yes' : 'No'),
    },
    {
      key: 'hubspotDefined',
      header: 'Built-in',
      render: (row) =>
        row.hubspotDefined ? (
          <Badge variant="secondary">Built-in</Badge>
        ) : (
          <Badge variant="outline">Custom</Badge>
        ),
    },
    {
      key: '_actions',
      header: '',
      render: (row) => (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm" className="h-7 w-7 p-0">
              <Settings2 className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => openEdit(row as unknown as Property)}>
              <Pencil className="mr-2 h-4 w-4" />
              Edit
            </DropdownMenuItem>
            <DropdownMenuItem
              variant="destructive"
              onClick={() => setConfirmArchive(row.name as string)}
            >
              <Trash2 className="mr-2 h-4 w-4" />
              Archive
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ),
    },
  ]

  const availableFieldTypes = FIELD_TYPES_BY_TYPE[form.type] ?? ['text']

  return (
    <div className="space-y-6 p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Properties</h1>
          <p className="text-sm text-muted-foreground">
            Manage properties for your CRM objects
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Select value={objectType} onValueChange={setObjectType}>
            <SelectTrigger className="w-[160px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {OBJECT_TYPES.map((t) => (
                <SelectItem key={t} value={t}>
                  {t.charAt(0).toUpperCase() + t.slice(1)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button onClick={openCreate}>
            <Plus className="mr-2 h-4 w-4" />
            Create Property
          </Button>
        </div>
      </div>

      {!isLoading && properties.length === 0 ? (
        <EmptyState
          icon={Settings2}
          title="No properties"
          description={`No properties found for ${objectType}.`}
          actionLabel="Create Property"
          onAction={openCreate}
        />
      ) : (
        <DataTable
          columns={columns}
          data={properties as (Property & Record<string, unknown>)[]}
          loading={isLoading}
          rowKey={(row) => row.name as string}
        />
      )}

      {/* Create / Edit Dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-lg max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>
              {editingProperty ? 'Edit Property' : 'Create Property'}
            </DialogTitle>
            <DialogDescription>
              {editingProperty
                ? `Editing property "${editingProperty}"`
                : `Create a new property for ${objectType}`}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            {!editingProperty && (
              <div className="space-y-2">
                <Label>Name</Label>
                <Input
                  value={form.name}
                  onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                  placeholder="property_name"
                />
              </div>
            )}

            <div className="space-y-2">
              <Label>Label</Label>
              <Input
                value={form.label}
                onChange={(e) => setForm((f) => ({ ...f, label: e.target.value }))}
                placeholder="Property Label"
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Type</Label>
                <Select value={form.type} onValueChange={handleTypeChange}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {PROPERTY_TYPES.map((t) => (
                      <SelectItem key={t} value={t}>
                        {t}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>Field Type</Label>
                <Select
                  value={form.fieldType}
                  onValueChange={(v) => setForm((f) => ({ ...f, fieldType: v }))}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {availableFieldTypes.map((ft) => (
                      <SelectItem key={ft} value={ft}>
                        {ft}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>

            <div className="space-y-2">
              <Label>Group Name</Label>
              <Input
                value={form.groupName}
                onChange={(e) => setForm((f) => ({ ...f, groupName: e.target.value }))}
                placeholder="contactinformation"
              />
            </div>

            <div className="space-y-2">
              <Label>Description</Label>
              <Input
                value={form.description}
                onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
                placeholder="A description of this property"
              />
            </div>

            {form.type === 'enumeration' && (
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <Label>Options</Label>
                  <Button variant="outline" size="sm" onClick={addOption}>
                    <Plus className="mr-1 h-3 w-3" />
                    Add Option
                  </Button>
                </div>
                {form.options.map((opt, idx) => (
                  <div key={idx} className="flex items-center gap-2">
                    <Input
                      className="flex-1"
                      value={opt.label}
                      onChange={(e) => updateOption(idx, 'label', e.target.value)}
                      placeholder="Label"
                    />
                    <Input
                      className="flex-1"
                      value={opt.value}
                      onChange={(e) => updateOption(idx, 'value', e.target.value)}
                      placeholder="Value"
                    />
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-8 w-8 p-0"
                      onClick={() => removeOption(idx)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleSubmit}
              disabled={createMutation.isPending || updateMutation.isPending}
            >
              {editingProperty ? 'Update' : 'Create'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Archive Confirmation Dialog */}
      <Dialog open={confirmArchive !== null} onOpenChange={() => setConfirmArchive(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Archive Property</DialogTitle>
            <DialogDescription>
              Are you sure you want to archive the property &quot;{confirmArchive}&quot;? This
              action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirmArchive(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={() => confirmArchive && handleArchive(confirmArchive)}
              disabled={archiveMutation.isPending}
            >
              Archive
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
