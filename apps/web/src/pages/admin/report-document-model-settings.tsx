import { Loader2, Rocket, Settings2 } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'

import { InlineNotice } from '@/components/common'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectItemText,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useModelProfiles } from '@/features/admin-config'
import {
  formatReportGatewayError,
  useReportSettingsQuery,
  useUpdateReportSettingsMutation,
} from '@/features/reports'

export function ReportDocumentModelSettingsPage() {
  const [documentProfileId, setDocumentProfileId] = useState('')
  const [documentProfileTouched, setDocumentProfileTouched] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)

  const reportSettingsQuery = useReportSettingsQuery()
  const chatProfilesQuery = useModelProfiles('chat', true)
  const updateReportSettingsMutation = useUpdateReportSettingsMutation()

  const chatProfiles = useMemo(() => chatProfilesQuery.data ?? [], [chatProfilesQuery.data])
  const configuredDocumentProfileId = reportSettingsQuery.data?.llm?.profileId ?? ''
  const configuredDocumentModel = reportSettingsQuery.data?.llm?.model ?? ''
  const selectedDocumentProfile = chatProfiles.find((profile) => profile.id === documentProfileId)
  const selectedDocumentModel =
    selectedDocumentProfile?.model ??
    (documentProfileId === configuredDocumentProfileId ? configuredDocumentModel : '')
  const firstChatProfileId = chatProfiles[0]?.id ?? ''
  const showDocumentProfileFallback =
    documentProfileId.trim() !== '' &&
    !selectedDocumentProfile &&
    documentProfileId === configuredDocumentProfileId
  const hasDocumentProfileOptions = showDocumentProfileFallback || chatProfiles.length > 0
  const noDocumentProfileOptions = !chatProfilesQuery.isLoading && !hasDocumentProfileOptions

  useEffect(() => {
    if (documentProfileTouched) return
    if (!reportSettingsQuery.isSuccess) {
      setDocumentProfileId('')
      return
    }

    setDocumentProfileId(configuredDocumentProfileId || firstChatProfileId)
  }, [
    configuredDocumentProfileId,
    documentProfileTouched,
    firstChatProfileId,
    reportSettingsQuery.isSuccess,
  ])

  const handleSelectDocumentProfile = (profileId: string) => {
    setDocumentProfileTouched(true)
    setDocumentProfileId(profileId)
    setNotice(null)
  }

  const handlePublishDocumentProfile = async () => {
    const profileId = documentProfileId.trim()
    setNotice(null)

    if (reportSettingsQuery.isLoading) {
      setNotice('正在读取当前文档生成模型配置，请稍后再发布。')
      return
    }
    if (!profileId) {
      setNotice('请选择用于报告生成的文档生成模型。')
      return
    }

    try {
      await updateReportSettingsMutation.mutateAsync({
        llm: { profileId, provider: 'ai-gateway' },
      })
      setNotice('文档生成模型配置已发布。')
    } catch (error) {
      setNotice(formatReportGatewayError(error, '文档生成模型配置发布失败'))
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="text-2xl font-semibold text-foreground">文档模型配置</h3>
          <p className="mt-1 text-sm text-muted-foreground">
            配置报告正文与大纲生成使用的 Document 侧模型 Profile。
          </p>
        </div>
      </div>

      <section className="rounded-lg border border-border bg-card p-4">
        <div className="flex items-center gap-2">
          <Settings2 className="size-4 text-muted-foreground" />
          <h2 className="text-base font-semibold">当前文档生成模型</h2>
        </div>

        <div className="mt-4 rounded-lg border border-border bg-background p-3 text-sm">
          <div className="mb-2 font-medium text-foreground">当前生效引用</div>
          <dl className="grid gap-2 md:grid-cols-[10rem_minmax(0,1fr)]">
            <dt className="text-muted-foreground">服务</dt>
            <dd className="text-foreground">ai-gateway</dd>
            <dt className="text-muted-foreground">Profile ID</dt>
            <dd className="break-all text-foreground">{configuredDocumentProfileId || '-'}</dd>
            <dt className="text-muted-foreground">模型</dt>
            <dd className="break-all text-foreground">{configuredDocumentModel || '-'}</dd>
          </dl>
        </div>

        <label className="mt-4 block max-w-xl space-y-1.5 text-sm">
          <span className="font-medium text-foreground">文档生成模型</span>
          <Select
            value={documentProfileId || undefined}
            onValueChange={(value) => handleSelectDocumentProfile(String(value))}
            disabled={
              reportSettingsQuery.isLoading ||
              chatProfilesQuery.isLoading ||
              !hasDocumentProfileOptions
            }
          >
            <SelectTrigger className="h-8 w-full" aria-label="文档生成模型">
              <SelectValue
                placeholder={
                  noDocumentProfileOptions ? '请先创建聊天模型 Profile' : '请选择聊天模型 Profile'
                }
              />
            </SelectTrigger>
            <SelectContent>
              {showDocumentProfileFallback && (
                <SelectItem value={documentProfileId}>
                  <SelectItemText>
                    当前配置：{selectedDocumentModel || documentProfileId}
                  </SelectItemText>
                </SelectItem>
              )}
              {chatProfiles.map((profile) => (
                <SelectItem key={profile.id} value={profile.id}>
                  <SelectItemText>
                    {profile.name} / {profile.model}
                  </SelectItemText>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </label>

        <div className="mt-4 grid gap-2 text-sm md:max-w-xl md:grid-cols-[10rem_minmax(0,1fr)]">
          <span className="text-muted-foreground">待发布 Profile</span>
          <code className="break-all">{documentProfileId || '-'}</code>
          <span className="text-muted-foreground">待发布模型</span>
          <span className="break-all">{selectedDocumentModel || '-'}</span>
        </div>

        {chatProfilesQuery.isError && (
          <InlineNotice className="mt-3" title="模型列表加载失败" variant="error">
            {formatReportGatewayError(chatProfilesQuery.error, '模型列表加载失败')}
          </InlineNotice>
        )}
        {reportSettingsQuery.isError && (
          <InlineNotice className="mt-3" title="报告设置加载失败" variant="error">
            {formatReportGatewayError(reportSettingsQuery.error, '报告设置加载失败')}
          </InlineNotice>
        )}
        {!chatProfilesQuery.isLoading && chatProfiles.length === 0 && (
          <InlineNotice className="mt-3" title="暂无可用模型" variant="warning">
            请先在模型管理中新增并启用用途为 chat 的模型 Profile。
          </InlineNotice>
        )}
        {notice && (
          <InlineNotice
            className="mt-3"
            title={notice.includes('失败') ? '发布失败' : undefined}
            variant={notice.includes('失败') ? 'error' : 'info'}
          >
            {notice}
          </InlineNotice>
        )}

        <Button
          type="button"
          className="mt-4"
          onClick={() => void handlePublishDocumentProfile()}
          disabled={
            !documentProfileId.trim() ||
            reportSettingsQuery.isLoading ||
            updateReportSettingsMutation.isPending
          }
        >
          {updateReportSettingsMutation.isPending ? (
            <Loader2 className="size-4 animate-spin" />
          ) : (
            <Rocket className="size-4" />
          )}
          发布文档模型配置
        </Button>
      </section>
    </div>
  )
}
