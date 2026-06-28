import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'

import { getLLMConfig, testLLMConnection, updateLLMConfig } from '@/api/admin'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

interface FormData {
  api_url: string
  api_key: string
  model_name: string
  timeout: number
}

interface NotificationState {
  type: 'success' | 'error'
  text: string
}

export function SystemSettings() {
  const queryClient = useQueryClient()
  const [notification, setNotification] = useState<NotificationState | null>(null)
  const [formInitialized, setFormInitialized] = useState(false)

  // Draft form state
  const [form, setForm] = useState<FormData>({
    api_url: '',
    api_key: '',
    model_name: '',
    timeout: 30,
  })

  // Fetch current config
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['admin', 'llm-config'],
    queryFn: getLLMConfig,
    staleTime: 30_000,
  })

  // Sync form from fetched data (only on first load)
  useEffect(() => {
    if (data && !formInitialized) {
      setForm({
        api_url: data.api_url ?? '',
        api_key: data.api_key ?? '',
        model_name: data.model_name ?? '',
        timeout: data.timeout ?? 30,
      })
      setFormInitialized(true)
    }
  }, [data, formInitialized])

  // Notification auto-dismiss
  useEffect(() => {
    if (!notification) return
    const timer = setTimeout(() => setNotification(null), 4000)
    return () => clearTimeout(timer)
  }, [notification])

  // Save mutation
  const saveMutation = useMutation({
    mutationFn: (payload: FormData) => updateLLMConfig(payload),
    onSuccess: () => {
      setNotification({ type: 'success', text: '配置已保存' })
      queryClient.invalidateQueries({ queryKey: ['admin', 'llm-config'] })
    },
    onError: (err: Error) => {
      setNotification({ type: 'error', text: `保存失败: ${err.message}` })
    },
  })

  // Test connection mutation
  const testMutation = useMutation({
    mutationFn: (payload: { api_url: string; api_key: string; model_name: string }) =>
      testLLMConnection(payload),
    onSuccess: (res) => {
      setNotification({
        type: 'success',
        text: `连接成功！延迟 ${res.latency_ms}ms，模型 ${res.model}`,
      })
    },
    onError: (err: Error) => {
      setNotification({ type: 'error', text: `连接测试失败: ${err.message}` })
    },
  })

  const updateField = useCallback((field: keyof FormData, value: string | number) => {
    setForm((prev) => ({ ...prev, [field]: value }))
  }, [])

  const handleSave = () => {
    saveMutation.mutate(form)
  }

  const handleTest = () => {
    testMutation.mutate({
      api_url: form.api_url,
      api_key: form.api_key,
      model_name: form.model_name,
    })
  }

  // Loading state
  if (isLoading) {
    return (
      <div>
        <h3 className="mb-4 text-2xl font-semibold text-foreground">系统设置</h3>
        <p className="mb-6 text-sm text-muted-foreground">
          全局系统配置，包括 LLM API 连接、向量数据库连接、系统参数等。
        </p>
        <div className="animate-pulse space-y-4 rounded-lg border border-border bg-card p-6">
          <div className="h-5 w-24 rounded bg-muted" />
          <div className="grid grid-cols-2 gap-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i}>
                <div className="mb-1 h-4 w-16 rounded bg-muted" />
                <div className="h-8 w-full rounded bg-muted" />
              </div>
            ))}
          </div>
          <div className="flex gap-2">
            <div className="h-8 w-24 rounded bg-muted" />
            <div className="h-8 w-24 rounded bg-muted" />
          </div>
        </div>
      </div>
    )
  }

  // Error state
  if (isError) {
    return (
      <div>
        <h3 className="mb-4 text-2xl font-semibold text-foreground">系统设置</h3>
        <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-6 text-center">
          <p className="text-sm text-destructive">
            加载配置失败: {error instanceof Error ? error.message : '未知错误'}
          </p>
        </div>
      </div>
    )
  }

  return (
    <div>
      <h3 className="mb-4 text-2xl font-semibold text-foreground">系统设置</h3>
      <p className="mb-6 text-sm text-muted-foreground">
        全局系统配置，包括 LLM API 连接、向量数据库连接、系统参数等。
      </p>

      {/* Notification banner */}
      {notification && (
        <div
          role="alert"
          className={`mb-4 rounded-lg border px-4 py-3 text-sm ${
            notification.type === 'success'
              ? 'border-emerald-500/50 bg-emerald-50 text-emerald-800 dark:border-emerald-400/30 dark:bg-emerald-950 dark:text-emerald-300'
              : 'border-destructive/50 bg-destructive/10 text-destructive'
          }`}
        >
          {notification.text}
        </div>
      )}

      {/* LLM Config Form */}
      <div className="rounded-lg border border-border bg-card p-6">
        <h4 className="mb-5 text-lg font-semibold text-foreground">LLM 配置</h4>

        <div className="grid grid-cols-2 gap-4">
          {/* API URL */}
          <div>
            <label
              htmlFor="llm-api-url"
              className="mb-1.5 block text-sm font-medium text-foreground"
            >
              API 地址
            </label>
            <Input
              id="llm-api-url"
              type="text"
              placeholder="https://api.openai.com/v1"
              value={form.api_url}
              onChange={(e) => updateField('api_url', e.target.value)}
            />
          </div>

          {/* Model Name */}
          <div>
            <label
              htmlFor="llm-model-name"
              className="mb-1.5 block text-sm font-medium text-foreground"
            >
              模型名称
            </label>
            <Input
              id="llm-model-name"
              type="text"
              placeholder="gpt-4o"
              value={form.model_name}
              onChange={(e) => updateField('model_name', e.target.value)}
            />
          </div>

          {/* API Key */}
          <div>
            <label
              htmlFor="llm-api-key"
              className="mb-1.5 block text-sm font-medium text-foreground"
            >
              API 密钥
            </label>
            <Input
              id="llm-api-key"
              type="password"
              placeholder="sk-••••••••"
              value={form.api_key}
              onChange={(e) => updateField('api_key', e.target.value)}
            />
          </div>

          {/* Timeout */}
          <div>
            <label
              htmlFor="llm-timeout"
              className="mb-1.5 block text-sm font-medium text-foreground"
            >
              超时时间（秒）
            </label>
            <Input
              id="llm-timeout"
              type="number"
              placeholder="30"
              min={1}
              max={300}
              value={form.timeout}
              onChange={(e) => updateField('timeout', Number(e.target.value))}
            />
          </div>
        </div>

        {/* Action buttons */}
        <div className="mt-5 flex gap-2">
          <Button onClick={handleSave} disabled={saveMutation.isPending}>
            {saveMutation.isPending && (
              <Loader2 aria-hidden="true" className="mr-1.5 size-3.5 animate-spin" />
            )}
            保存配置
          </Button>
          <Button variant="outline" onClick={handleTest} disabled={testMutation.isPending}>
            {testMutation.isPending && (
              <Loader2 aria-hidden="true" className="mr-1.5 size-3.5 animate-spin" />
            )}
            测试连接
          </Button>
        </div>
      </div>
    </div>
  )
}
