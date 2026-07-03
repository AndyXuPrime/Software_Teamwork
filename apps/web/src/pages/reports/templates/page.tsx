import { FileText, Trash2, Upload } from 'lucide-react'
import { type FormEvent, useEffect, useMemo, useState } from 'react'

import { InlineNotice, StateBlock } from '@/components/common'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectItemText,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import type { ReportTemplate } from '@/features/reports'
import {
  formatReportGatewayError,
  useCreateTemplate,
  useDeleteTemplate,
  useReportBootstrapQueries,
  useReportStatisticsQueries,
  useTemplateStructure,
  useUpdateTemplateStructure,
} from '@/features/reports'

type TemplateUploadForm = {
  description: string
  file: File | null
  reportType: string
  templateName: string
}

const emptyUploadForm: TemplateUploadForm = {
  description: '',
  file: null,
  reportType: '',
  templateName: '',
}

export function ReportTemplatesPage() {
  const [structureTarget, setStructureTarget] = useState<string | null>(null)
  const [editMode, setEditMode] = useState(false)
  const [editJson, setEditJson] = useState('')
  const [jsonError, setJsonError] = useState<string | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<ReportTemplate | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [uploadOpen, setUploadOpen] = useState(false)
  const [uploadForm, setUploadForm] = useState<TemplateUploadForm>(emptyUploadForm)
  const [uploadError, setUploadError] = useState<string | null>(null)
  const [uploadNotice, setUploadNotice] = useState<string | null>(null)

  const { typeQuery, templateQuery, materialQuery } = useReportBootstrapQueries()
  const { overviewQuery, dailyQuery } = useReportStatisticsQueries()
  const structureQuery = useTemplateStructure(structureTarget)
  const updateStructureMutation = useUpdateTemplateStructure(structureTarget ?? '')
  const createTemplateMutation = useCreateTemplate()
  const deleteMutation = useDeleteTemplate()

  const reportTypes = useMemo(() => typeQuery.data ?? [], [typeQuery.data])
  const templates = templateQuery.data?.items ?? []
  const materials = materialQuery.data?.items ?? []
  const overview = overviewQuery.data
  const daily = dailyQuery.data ?? []
  const queryErrors = [
    { error: typeQuery.error, label: '报告类型', visible: typeQuery.isError },
    { error: templateQuery.error, label: '模板列表', visible: templateQuery.isError },
    { error: materialQuery.error, label: '素材列表', visible: materialQuery.isError },
    { error: overviewQuery.error, label: '统计概览', visible: overviewQuery.isError },
    { error: dailyQuery.error, label: '统计趋势', visible: dailyQuery.isError },
  ].filter((item) => item.visible)

  useEffect(() => {
    if (!uploadOpen || uploadForm.reportType || reportTypes.length === 0) return
    setUploadForm((prev) => ({ ...prev, reportType: reportTypes[0]?.code ?? '' }))
  }, [reportTypes, uploadForm.reportType, uploadOpen])

  const resetUploadDialog = () => {
    setUploadForm(emptyUploadForm)
    setUploadError(null)
  }

  const handleUploadOpenChange = (open: boolean) => {
    setUploadOpen(open)
    if (!open) resetUploadDialog()
  }

  const handleUploadSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setUploadError(null)
    setUploadNotice(null)

    const templateName = uploadForm.templateName.trim()
    const reportType = uploadForm.reportType.trim()
    const description = uploadForm.description.trim()

    if (!templateName) {
      setUploadError('请输入模板名称。')
      return
    }
    if (!reportType) {
      setUploadError('请选择报告类型。')
      return
    }
    if (!uploadForm.file) {
      setUploadError('请选择要上传的模板文件。')
      return
    }

    createTemplateMutation.mutate(
      {
        description: description || undefined,
        file: uploadForm.file,
        reportType,
        templateName,
      },
      {
        onError: (error) => setUploadError(formatReportGatewayError(error, '上传模板失败')),
        onSuccess: () => {
          setUploadNotice('模板上传成功，列表已刷新。')
          setUploadOpen(false)
          resetUploadDialog()
        },
      },
    )
  }

  const handleOpenStructure = (templateId: string) => {
    setStructureTarget(templateId)
    setEditMode(false)
    setJsonError(null)
  }

  const handleCloseStructure = () => {
    setStructureTarget(null)
    setEditMode(false)
    setJsonError(null)
  }

  const handleEnterEdit = () => {
    const data = structureQuery.data
    if (data) {
      setEditJson(JSON.stringify(data, null, 2))
      setEditMode(true)
      setJsonError(null)
    }
  }

  const handleSaveEdit = () => {
    try {
      const parsed = JSON.parse(editJson) as Record<string, unknown>
      if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
        setJsonError('模板结构必须是一个 JSON 对象')
        return
      }
      setJsonError(null)
      updateStructureMutation.mutate(
        parsed as Parameters<typeof updateStructureMutation.mutate>[0],
        {
          onSuccess: () => setEditMode(false),
          onError: (error) => setJsonError(formatReportGatewayError(error, '保存失败，请重试')),
        },
      )
    } catch {
      setJsonError('JSON 格式无效，请检查语法')
    }
  }

  const handleCancelEdit = () => {
    setEditMode(false)
    setJsonError(null)
  }

  const handleDelete = () => {
    if (!deleteTarget) return
    setDeleteError(null)
    deleteMutation.mutate(deleteTarget.id, {
      onSuccess: () => setDeleteTarget(null),
      onError: (error) => setDeleteError(formatReportGatewayError(error, '删除模板失败')),
    })
  }

  const structureData = structureQuery.data
  const structureJson = structureData ? JSON.stringify(structureData, null, 2) : ''

  return (
    <div className="h-full overflow-auto bg-background p-6">
      <div className="mb-5 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">报告模板与素材</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            管理员能力入口：模板、素材、结构配置、统计和任务诊断。
          </p>
        </div>
        <div className="flex gap-2">
          <Button disabled title="上传素材表单尚未接入当前页面" variant="outline">
            <Upload className="size-4" />
            上传素材
          </Button>
          <Button onClick={() => setUploadOpen(true)}>
            <Upload className="size-4" />
            上传模板
          </Button>
        </div>
      </div>

      {uploadNotice && (
        <InlineNotice className="mb-4" title="上传完成" variant="success">
          {uploadNotice}
        </InlineNotice>
      )}

      {queryErrors.map((item) => (
        <InlineNotice
          className="mb-3"
          key={item.label}
          title={`${item.label}加载失败`}
          variant="error"
        >
          {formatReportGatewayError(item.error, `${item.label}加载失败`)}
        </InlineNotice>
      ))}

      {(templateQuery.isError || materialQuery.isError) && (
        <InlineNotice className="mb-4" title="能力边界" variant="warning">
          页面不会使用本地模板或素材示例兜底；请以 Gateway Document API 返回结果为准。
        </InlineNotice>
      )}

      <div className="mb-6 grid gap-4 md:grid-cols-3">
        <section className="rounded-lg border border-border bg-card p-4 hover:-translate-y-0.5 hover:shadow-md transition-all duration-200">
          <p className="text-sm text-muted-foreground">模板数量</p>
          <p className="mt-2 text-2xl font-semibold">{overview?.templateCount ?? '-'}</p>
        </section>
        <section className="rounded-lg border border-border bg-card p-4 hover:-translate-y-0.5 hover:shadow-md transition-all duration-200">
          <p className="text-sm text-muted-foreground">素材数量</p>
          <p className="mt-2 text-2xl font-semibold">{overview?.materialCount ?? '-'}</p>
        </section>
        <section className="rounded-lg border border-border bg-card p-4 hover:-translate-y-0.5 hover:shadow-md transition-all duration-200">
          <p className="text-sm text-muted-foreground">近 30 天报告</p>
          <p className="mt-2 text-2xl font-semibold">
            {overview?.reportCount ??
              (dailyQuery.isSuccess
                ? daily.reduce((total, item) => total + item.createdCount, 0)
                : '-')}
          </p>
        </section>
      </div>

      <div className="grid gap-6 xl:grid-cols-2">
        <section className="rounded-lg border border-border bg-card">
          <div className="border-b border-border px-4 py-3">
            <h2 className="flex items-center gap-2 text-base font-semibold">
              <FileText className="size-4" />
              模板列表
            </h2>
          </div>
          <div className="divide-y divide-border">
            {templateQuery.isLoading ? (
              <StateBlock size="compact" title="模板加载中" variant="loading" />
            ) : templateQuery.isError ? (
              <StateBlock
                description={formatReportGatewayError(templateQuery.error, '模板列表加载失败')}
                size="compact"
                title="模板列表加载失败"
                variant="error"
              />
            ) : templates.length === 0 ? (
              <StateBlock size="compact" title="暂无报告模板" variant="empty" />
            ) : (
              templates.map((template) => (
                <div
                  key={template.id}
                  className="flex items-center justify-between gap-4 p-4 hover:bg-muted/20 transition-colors"
                >
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium">{template.templateName}</p>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {template.reportType} · v{template.version} · {template.filename}
                    </p>
                  </div>
                  <div className="flex items-center gap-1.5 shrink-0">
                    <Button
                      variant="outline"
                      size="xs"
                      onClick={() => handleOpenStructure(template.id)}
                    >
                      查看结构
                    </Button>
                    <span className="rounded-full bg-muted px-2 py-1 text-xs">
                      {template.enabled ? '启用' : '停用'}
                    </span>
                    <Button
                      variant="ghost"
                      size="icon-xs"
                      aria-label="删除模板"
                      onClick={() => setDeleteTarget(template)}
                    >
                      <Trash2 className="size-3 text-destructive" />
                    </Button>
                  </div>
                </div>
              ))
            )}
          </div>
        </section>

        <section className="rounded-lg border border-border bg-card">
          <div className="border-b border-border px-4 py-3">
            <h2 className="flex items-center gap-2 text-base font-semibold">
              <FileText className="size-4" />
              专业素材
            </h2>
          </div>
          <div className="divide-y divide-border">
            {materialQuery.isLoading ? (
              <StateBlock size="compact" title="素材加载中" variant="loading" />
            ) : materialQuery.isError ? (
              <StateBlock
                description={formatReportGatewayError(materialQuery.error, '素材列表加载失败')}
                size="compact"
                title="素材列表加载失败"
                variant="error"
              />
            ) : materials.length === 0 ? (
              <StateBlock size="compact" title="暂无报告素材" variant="empty" />
            ) : (
              materials.map((material) => (
                <div
                  key={material.id}
                  className="flex items-center justify-between gap-4 p-4 hover:bg-muted/20 transition-colors"
                >
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium">{material.materialName}</p>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {material.category ?? '-'} · {material.materialType ?? 'material'}
                    </p>
                  </div>
                  <span className="rounded-full bg-muted px-2 py-1 text-xs">
                    {material.enabled ? '可引用' : '停用'}
                  </span>
                </div>
              ))
            )}
          </div>
        </section>
      </div>

      {/* Template upload dialog */}
      <Dialog open={uploadOpen} onOpenChange={handleUploadOpenChange}>
        <DialogContent className="sm:max-w-lg">
          <form onSubmit={handleUploadSubmit}>
            <DialogHeader>
              <DialogTitle>上传报告模板</DialogTitle>
              <DialogDescription>上传 DOCX 模板并关联报告类型。</DialogDescription>
            </DialogHeader>

            <div className="mt-4 grid gap-3">
              <label className="grid gap-1.5 text-sm">
                <span className="font-medium">模板名称</span>
                <Input
                  value={uploadForm.templateName}
                  onChange={(event) =>
                    setUploadForm((prev) => ({ ...prev, templateName: event.target.value }))
                  }
                  placeholder="例如：迎峰度夏巡检模板"
                />
              </label>

              <label className="grid gap-1.5 text-sm">
                <span className="font-medium">报告类型</span>
                <Select
                  value={uploadForm.reportType || undefined}
                  onValueChange={(value) =>
                    setUploadForm((prev) => ({ ...prev, reportType: value }))
                  }
                  disabled={typeQuery.isLoading || reportTypes.length === 0}
                >
                  <SelectTrigger aria-label="报告类型">
                    <SelectValue
                      placeholder={typeQuery.isLoading ? '加载中...' : '请选择报告类型'}
                    />
                  </SelectTrigger>
                  <SelectContent>
                    {reportTypes.map((type) => (
                      <SelectItem key={type.code} value={type.code}>
                        <SelectItemText>
                          {type.name} / {type.code}
                        </SelectItemText>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </label>

              <label className="grid gap-1.5 text-sm">
                <span className="font-medium">模板文件</span>
                <Input
                  accept=".doc,.docx,application/msword,application/vnd.openxmlformats-officedocument.wordprocessingml.document"
                  type="file"
                  onChange={(event) =>
                    setUploadForm((prev) => ({
                      ...prev,
                      file: event.target.files?.[0] ?? null,
                    }))
                  }
                />
              </label>

              <label className="grid gap-1.5 text-sm">
                <span className="font-medium">描述</span>
                <Textarea
                  value={uploadForm.description}
                  onChange={(event) =>
                    setUploadForm((prev) => ({ ...prev, description: event.target.value }))
                  }
                  placeholder="可选，用于说明模板适用场景。"
                />
              </label>

              {uploadError && (
                <InlineNotice title="上传失败" variant="error">
                  {uploadError}
                </InlineNotice>
              )}
              {!typeQuery.isLoading && reportTypes.length === 0 && (
                <InlineNotice title="暂无报告类型" variant="warning">
                  请先确认报告类型接口已返回可用类型。
                </InlineNotice>
              )}
            </div>

            <DialogFooter className="mt-5">
              <Button type="button" variant="outline" onClick={() => handleUploadOpenChange(false)}>
                取消
              </Button>
              <Button type="submit" disabled={createTemplateMutation.isPending}>
                {createTemplateMutation.isPending ? '上传中...' : '上传'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Template structure viewer / editor dialog */}
      <Dialog
        open={Boolean(structureTarget)}
        onOpenChange={(open) => !open && handleCloseStructure()}
      >
        <DialogContent className="sm:max-w-xl">
          <DialogHeader>
            <DialogTitle>
              {structureTarget
                ? `模板结构 - ${templates.find((t) => t.id === structureTarget)?.templateName ?? structureTarget}`
                : '模板结构'}
            </DialogTitle>
            <DialogDescription>
              {editMode
                ? '编辑模板的 outlineSchema 和 styleConfig 配置。'
                : '模板的 JSON 结构定义。'}
            </DialogDescription>
          </DialogHeader>

          {structureQuery.isLoading && (
            <div className="py-4 text-center text-sm text-muted-foreground">加载中...</div>
          )}

          {structureQuery.isError && (
            <div className="py-4 text-center text-sm text-destructive">
              {formatReportGatewayError(structureQuery.error, '模板结构加载失败')}
            </div>
          )}

          {!structureQuery.isLoading && !structureQuery.isError && (
            <>
              {editMode ? (
                <div className="flex flex-col gap-2">
                  <Textarea
                    className="min-h-80 font-mono text-xs"
                    value={editJson}
                    onChange={(event) => {
                      setEditJson(event.target.value)
                      setJsonError(null)
                    }}
                    placeholder='{"outlineSchema": [...], "styleConfig": {...}}'
                  />
                  {jsonError && <p className="text-xs text-destructive">{jsonError}</p>}
                </div>
              ) : (
                <pre className="max-h-96 overflow-auto rounded-lg bg-muted p-4 font-mono text-xs leading-relaxed">
                  {structureJson || '{}'}
                </pre>
              )}
            </>
          )}

          <DialogFooter>
            {!editMode ? (
              <>
                <Button variant="outline" onClick={handleCloseStructure}>
                  关闭
                </Button>
                {structureTarget && (
                  <Button onClick={handleEnterEdit} disabled={structureQuery.isError}>
                    编辑结构
                  </Button>
                )}
              </>
            ) : (
              <>
                <Button variant="outline" onClick={handleCancelEdit}>
                  取消
                </Button>
                <Button onClick={handleSaveEdit} disabled={updateStructureMutation.isPending}>
                  {updateStructureMutation.isPending ? '保存中...' : '保存'}
                </Button>
              </>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete template confirmation dialog */}
      <Dialog
        open={Boolean(deleteTarget)}
        onOpenChange={(open) => {
          if (!open) {
            setDeleteTarget(null)
            setDeleteError(null)
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确定删除此模板？</DialogTitle>
            <DialogDescription>
              <span>
                {deleteTarget?.templateName
                  ? `即将删除模板"${deleteTarget.templateName}"。此操作不可撤销。`
                  : '此操作不可撤销。'}
              </span>
              {deleteError && <span className="mt-2 block text-destructive">{deleteError}</span>}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setDeleteTarget(null)
                setDeleteError(null)
              }}
              disabled={deleteMutation.isPending}
            >
              取消
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? '删除中...' : '确认删除'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
