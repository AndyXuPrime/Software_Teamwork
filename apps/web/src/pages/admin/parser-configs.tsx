import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import {
  createParserConfig,
  deleteParserConfig,
  listParserConfigs,
  updateParserConfig,
} from '@/api/admin'
import { ApiError } from '@/api/client'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import type {
  CreateParserConfigRequest,
  ParserBackend,
  ParserConfig,
  UpdateParserConfigRequest,
} from '@/lib/types'

const parserBackends: ParserBackend[] = [
  'builtin',
  'tika',
  'unstructured',
  'local_ocr',
  'remote_compatible',
]

type FormState = {
  name: string
  backend: ParserBackend
  enabled: boolean
  isDefault: boolean
  concurrency: string
  supportedContentTypes: string
  endpointUrl: string
  defaultParameters: string
}

const emptyForm: FormState = {
  name: '',
  backend: 'builtin',
  enabled: true,
  isDefault: false,
  concurrency: '4',
  supportedContentTypes: 'application/pdf,text/plain',
  endpointUrl: '',
  defaultParameters: '{}',
}

function errorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    return `${error.code}: ${error.message}`
  }
  if (error instanceof Error) {
    return error.message
  }
  return ''
}

function formFromConfig(config: ParserConfig): FormState {
  return {
    name: config.name,
    backend: config.backend,
    enabled: config.enabled,
    isDefault: config.isDefault,
    concurrency: String(config.concurrency),
    supportedContentTypes: (config.supportedContentTypes ?? []).join(','),
    endpointUrl: config.endpointUrl ?? '',
    defaultParameters: JSON.stringify(config.defaultParameters ?? {}, null, 2),
  }
}

function parseForm(form: FormState): CreateParserConfigRequest | string {
  let defaultParameters: Record<string, unknown>
  try {
    const parsed: unknown = JSON.parse(form.defaultParameters)
    if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') {
      return 'Default parameters must be a JSON object.'
    }
    defaultParameters = parsed as Record<string, unknown>
  } catch {
    return 'Default parameters must be valid JSON.'
  }

  const concurrency = Number(form.concurrency)
  if (!Number.isInteger(concurrency) || concurrency < 1 || concurrency > 128) {
    return 'Concurrency must be an integer between 1 and 128.'
  }
  if (form.backend === 'remote_compatible' && !form.endpointUrl.trim()) {
    return 'Remote compatible backend requires an endpoint URL.'
  }
  if (!form.name.trim()) {
    return 'Name is required.'
  }

  return {
    name: form.name.trim(),
    backend: form.backend,
    enabled: form.enabled,
    isDefault: form.isDefault,
    concurrency,
    supportedContentTypes: form.supportedContentTypes
      .split(',')
      .map((value) => value.trim())
      .filter(Boolean),
    endpointUrl: form.endpointUrl.trim() || null,
    defaultParameters,
  }
}

export function ParserConfigsPage() {
  const queryClient = useQueryClient()
  const [editing, setEditing] = useState<ParserConfig | null>(null)
  const [form, setForm] = useState<FormState>(emptyForm)
  const [formError, setFormError] = useState('')

  const configsQuery = useQuery({
    queryKey: ['admin', 'parser-configs'],
    queryFn: () => listParserConfigs(),
  })

  const saveMutation = useMutation({
    mutationFn: (input: CreateParserConfigRequest | UpdateParserConfigRequest) =>
      editing
        ? updateParserConfig(editing.id, input as UpdateParserConfigRequest)
        : createParserConfig(input as CreateParserConfigRequest),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin', 'parser-configs'] })
      setEditing(null)
      setForm(emptyForm)
      setFormError('')
    },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteParserConfig,
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['admin', 'parser-configs'] }),
  })

  const toggleMutation = useMutation({
    mutationFn: (config: ParserConfig) =>
      updateParserConfig(config.id, { enabled: !config.enabled }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['admin', 'parser-configs'] }),
  })

  const handleSubmit = () => {
    const parsed = parseForm(form)
    if (typeof parsed === 'string') {
      setFormError(parsed)
      return
    }
    setFormError('')
    saveMutation.mutate(parsed)
  }

  const handleEdit = (config: ParserConfig) => {
    setEditing(config)
    setForm(formFromConfig(config))
    setFormError('')
  }

  const handleToggle = (config: ParserConfig) => {
    toggleMutation.mutate(config)
  }

  const requestError = errorMessage(
    saveMutation.error ?? toggleMutation.error ?? deleteMutation.error ?? configsQuery.error,
  )

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-2xl font-semibold">Parser configs</h3>
        <p className="text-sm text-muted-foreground">
          Manage the runtime parser configuration used by new document processing jobs.
        </p>
      </div>

      {(formError || requestError) && (
        <div className="rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
          {formError || requestError}
        </div>
      )}

      <section className="grid gap-4 rounded-lg border bg-card p-5 md:grid-cols-2">
        <label className="text-sm">
          Name
          <Input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} />
        </label>
        <label className="text-sm">
          Backend
          <select
            className="mt-1 h-9 w-full rounded-md border bg-background px-3"
            value={form.backend}
            onChange={(event) => setForm({ ...form, backend: event.target.value as ParserBackend })}
          >
            {parserBackends.map((backend) => (
              <option key={backend} value={backend}>
                {backend}
              </option>
            ))}
          </select>
        </label>
        <label className="text-sm">
          Concurrency
          <Input
            max={128}
            min={1}
            type="number"
            value={form.concurrency}
            onChange={(event) => setForm({ ...form, concurrency: event.target.value })}
          />
        </label>
        <label className="text-sm">
          Supported content types
          <Input
            value={form.supportedContentTypes}
            onChange={(event) => setForm({ ...form, supportedContentTypes: event.target.value })}
          />
        </label>
        <label className="text-sm md:col-span-2">
          Endpoint URL
          <Input
            value={form.endpointUrl}
            onChange={(event) => setForm({ ...form, endpointUrl: event.target.value })}
          />
        </label>
        <label className="text-sm md:col-span-2">
          Default parameters JSON
          <Textarea
            className="min-h-32 font-mono"
            value={form.defaultParameters}
            onChange={(event) => setForm({ ...form, defaultParameters: event.target.value })}
          />
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input
            checked={form.enabled}
            type="checkbox"
            onChange={(event) => setForm({ ...form, enabled: event.target.checked })}
          />
          Enabled
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input
            checked={form.isDefault}
            type="checkbox"
            onChange={(event) => setForm({ ...form, isDefault: event.target.checked })}
          />
          Default config
        </label>
        <div className="flex gap-2 md:col-span-2">
          <Button disabled={saveMutation.isPending} onClick={handleSubmit}>
            {saveMutation.isPending ? 'Saving...' : editing ? 'Save changes' : 'Create config'}
          </Button>
          {editing && (
            <Button
              variant="outline"
              onClick={() => {
                setEditing(null)
                setForm(emptyForm)
              }}
            >
              Cancel
            </Button>
          )}
        </div>
      </section>

      <section className="rounded-lg border bg-card">
        {configsQuery.isLoading ? (
          <p className="p-6 text-sm text-muted-foreground">Loading...</p>
        ) : configsQuery.data?.length ? (
          <div className="divide-y">
            {configsQuery.data.map((config) => (
              <div key={config.id} className="flex flex-wrap items-center justify-between gap-3 p-4">
                <div>
                  <div className="flex items-center gap-2 font-medium">
                    {config.name}
                    {config.isDefault && <Badge>Default</Badge>}
                    {!config.enabled && <Badge variant="secondary">Disabled</Badge>}
                  </div>
                  <p className="text-xs text-muted-foreground">
                    {config.backend} · concurrency {config.concurrency} ·{' '}
                    {(config.supportedContentTypes ?? []).join(', ') || 'all content types'}
                  </p>
                </div>
                <div className="flex gap-2">
                  <Button size="sm" variant="outline" onClick={() => handleEdit(config)}>
                    Edit
                  </Button>
                  <Button
                    disabled={toggleMutation.isPending}
                    size="sm"
                    variant="outline"
                    onClick={() => handleToggle(config)}
                  >
                    {config.enabled ? 'Disable' : 'Enable'}
                  </Button>
                  <Button
                    disabled={config.isDefault || deleteMutation.isPending}
                    size="sm"
                    variant="destructive"
                    onClick={() => deleteMutation.mutate(config.id)}
                  >
                    Delete
                  </Button>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <p className="p-6 text-sm text-muted-foreground">No parser configs yet.</p>
        )}
      </section>
    </div>
  )
}
