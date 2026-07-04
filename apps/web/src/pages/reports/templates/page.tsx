import { Eye, FileText, Trash2, Upload } from 'lucide-react'
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
import type { ReportMaterial, ReportTemplate } from '@/features/reports'
import {
  formatReportGatewayError,
  getReportMaterialDisplayDetails,
  useCreateMaterial,
  useCreateTemplate,
  useDeleteMaterial,
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

type MaterialUploadForm = {
  category: string
  description: string
  file: File | null
  materialName: string
  materialType: string
  tags: string
}

const emptyUploadForm: TemplateUploadForm = {
  description: '',
  file: null,
  reportType: '',
  templateName: '',
}

const emptyMaterialUploadForm: MaterialUploadForm = {
  category: '',
  description: '',
  file: null,
  materialName: '',
  materialType: '',
  tags: '',
}

const reportTemplateAcceptedMime =
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document'
const reportTemplateFileAccept = `.docx,${reportTemplateAcceptedMime}`
const reportTemplateMaxUploadBytes = 32 * 1024 * 1024

function getTemplateFileValidationError(file: File): string | null {
  const filename = file.name.trim().toLowerCase()
  if (!filename.endsWith('.docx')) return '仅支持上传 DOCX 模板文件。'
  if (file.type && file.type !== reportTemplateAcceptedMime) {
    return '仅支持上传 DOCX 模板文件。'
  }
  if (file.size > reportTemplateMaxUploadBytes) return '模板文件不能超过 32 MiB。'
  return null
}

export function ReportTemplatesPage() {
  const [structureTarget, setStructureTarget] = useState<string | null>(null)
  const [editMode, setEditMode] = useState(false)
  const [editJson, setEditJson] = useState('')
  const [jsonError, setJsonError] = useState<string | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<ReportTemplate | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [materialViewTarget, setMaterialViewTarget] = useState<ReportMaterial | null>(null)
  const [materialDeleteTarget, setMaterialDeleteTarget] = useState<ReportMaterial | null>(null)
  const [materialDeleteError, setMaterialDeleteError] = useState<string | null>(null)
  const [uploadOpen, setUploadOpen] = useState(false)
  const [uploadForm, setUploadForm] = useState<TemplateUploadForm>(emptyUploadForm)
  const [uploadError, setUploadError] = useState<string | null>(null)
  const [uploadNotice, setUploadNotice] = useState<string | null>(null)
  const [materialUploadOpen, setMaterialUploadOpen] = useState(false)
  const [materialUploadForm, setMaterialUploadForm] =
    useState<MaterialUploadForm>(emptyMaterialUploadForm)
  const [materialUploadError, setMaterialUploadError] = useState<string | null>(null)

  const { typeQuery, templateQuery, materialQuery } = useReportBootstrapQueries()
  const { overviewQuery, dailyQuery } = useReportStatisticsQueries()
  const structureQuery = useTemplateStructure(structureTarget)
  const updateStructureMutation = useUpdateTemplateStructure(structureTarget ?? '')
  const createTemplateMutation = useCreateTemplate()
  const createMaterialMutation = useCreateMaterial()
  const deleteMutation = useDeleteTemplate()
  const deleteMaterialMutation = useDeleteMaterial()

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

  const resetMaterialUploadDialog = () => {
    setMaterialUploadForm(emptyMaterialUploadForm)
    setMaterialUploadError(null)
  }

  const handleMaterialUploadOpenChange = (open: boolean) => {
    setMaterialUploadOpen(open)
    if (!open) resetMaterialUploadDialog()
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
    const fileValidationError = getTemplateFileValidationError(uploadForm.file)
    if (fileValidationError) {
      setUploadError(fileValidationError)
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

  const handleMaterialUploadSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setMaterialUploadError(null)
    setUploadNotice(null)

    const materialName = materialUploadForm.materialName.trim()
    const materialType = materialUploadForm.materialType.trim()
    const category = materialUploadForm.category.trim()
    const description = materialUploadForm.description.trim()
    const tags = materialUploadForm.tags
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean)

    if (!materialName) {
      setMaterialUploadError('请输入素材名称。')
      return
    }
    if (!materialType) {
      setMaterialUploadError('请输入素材类型。')
      return
    }
    if (!materialUploadForm.file) {
      setMaterialUploadError('请选择要上传的素材文件。')
      return
    }

    createMaterialMutation.mutate(
      {
        category: category || undefined,
        description: description || undefined,
        file: materialUploadForm.file,
        materialName,
        materialType,
        tags,
      },
      {
        onError: (error) => setMaterialUploadError(formatReportGatewayError(error, '上传素材失败')),
        onSuccess: () => {
          setUploadNotice('素材上传成功，列表已刷新。')
          setMaterialUploadOpen(false)
          resetMaterialUploadDialog()
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

  const handleMaterialDelete = () => {
    if (!materialDeleteTarget) return
    setMaterialDeleteError(null)
    deleteMaterialMutation.mutate(materialDeleteTarget.id, {
      onSuccess: () => setMaterialDeleteTarget(null),
      onError: (error) => setMaterialDeleteError(formatReportGatewayError(error, '删除素材失败')),
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
          <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border px-4 py-3">
            <h2 className="flex items-center gap-2 text-base font-semibold">
              <FileText className="size-4" />
              模板列表
            </h2>
            <Button size="sm" onClick={() => setUploadOpen(true)}>
              <Upload className="size-3.5" />
              上传模板
            </Button>
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
          <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border px-4 py-3">
            <h2 className="flex items-center gap-2 text-base font-semibold">
              <FileText className="size-4" />
              专业素材
            </h2>
            <Button size="sm" onClick={() => setMaterialUploadOpen(true)}>
              <Upload className="size-3.5" />
              上传素材
            </Button>
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
              materials.map((material) => {
                const materialDisplay = getReportMaterialDisplayDetails(material)

                return (
                  <div
                    key={material.id}
                    className="flex items-center justify-between gap-4 p-4 hover:bg-muted/20 transition-colors"
                  >
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium">{materialDisplay.materialName}</p>
                    </div>
                    <div className="flex shrink-0 items-center gap-1.5">
                      <Button
                        variant="outline"
                        size="xs"
                        onClick={() => setMaterialViewTarget(material)}
                      >
                        <Eye className="size-3" />
                        查看素材
                      </Button>
                      <span className="rounded-full bg-muted px-2 py-1 text-xs">
                        {materialDisplay.enabledText}
                      </span>
                      <Button
                        variant="ghost"
                        size="icon-xs"
                        aria-label="删除素材"
                        onClick={() => setMaterialDeleteTarget(material)}
                      >
                        <Trash2 className="size-3 text-destructive" />
                      </Button>
                    </div>
                  </div>
                )
              })
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
                  accept={reportTemplateFileAccept}
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

      {/* Material upload dialog */}
      <Dialog open={materialUploadOpen} onOpenChange={handleMaterialUploadOpenChange}>
        <DialogContent className="sm:max-w-lg">
          <form onSubmit={handleMaterialUploadSubmit}>
            <DialogHeader>
              <DialogTitle>上传专业素材</DialogTitle>
              <DialogDescription>上传报告正文生成可引用的专业资料。</DialogDescription>
            </DialogHeader>

            <div className="mt-4 grid gap-3">
              <label className="grid gap-1.5 text-sm">
                <span className="font-medium">素材名称</span>
                <Input
                  value={materialUploadForm.materialName}
                  onChange={(event) =>
                    setMaterialUploadForm((prev) => ({
                      ...prev,
                      materialName: event.target.value,
                    }))
                  }
                  placeholder="例如：煤场盘点素材"
                />
              </label>

              <label className="grid gap-1.5 text-sm">
                <span className="font-medium">素材类型</span>
                <Input
                  value={materialUploadForm.materialType}
                  onChange={(event) =>
                    setMaterialUploadForm((prev) => ({
                      ...prev,
                      materialType: event.target.value,
                    }))
                  }
                  placeholder="例如：technical_doc"
                />
              </label>

              <label className="grid gap-1.5 text-sm">
                <span className="font-medium">分类</span>
                <Input
                  value={materialUploadForm.category}
                  onChange={(event) =>
                    setMaterialUploadForm((prev) => ({ ...prev, category: event.target.value }))
                  }
                  placeholder="可选，例如：煤库存"
                />
              </label>

              <label className="grid gap-1.5 text-sm">
                <span className="font-medium">标签</span>
                <Input
                  value={materialUploadForm.tags}
                  onChange={(event) =>
                    setMaterialUploadForm((prev) => ({ ...prev, tags: event.target.value }))
                  }
                  placeholder="可选，多个标签用英文逗号分隔"
                />
              </label>

              <label className="grid gap-1.5 text-sm">
                <span className="font-medium">素材文件</span>
                <Input
                  type="file"
                  onChange={(event) =>
                    setMaterialUploadForm((prev) => ({
                      ...prev,
                      file: event.target.files?.[0] ?? null,
                    }))
                  }
                />
              </label>

              <label className="grid gap-1.5 text-sm">
                <span className="font-medium">描述</span>
                <Textarea
                  value={materialUploadForm.description}
                  onChange={(event) =>
                    setMaterialUploadForm((prev) => ({
                      ...prev,
                      description: event.target.value,
                    }))
                  }
                  placeholder="可选，用于说明素材适用场景。"
                />
              </label>

              {materialUploadError && (
                <InlineNotice title="上传失败" variant="error">
                  {materialUploadError}
                </InlineNotice>
              )}
            </div>

            <DialogFooter className="mt-5">
              <Button
                type="button"
                variant="outline"
                onClick={() => handleMaterialUploadOpenChange(false)}
              >
                取消
              </Button>
              <Button type="submit" disabled={createMaterialMutation.isPending}>
                {createMaterialMutation.isPending ? '上传中...' : '上传'}
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
                    maxLength={50000}
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

      {/* Material detail dialog */}
      <Dialog
        open={Boolean(materialViewTarget)}
        onOpenChange={(open) => {
          if (!open) setMaterialViewTarget(null)
        }}
      >
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>素材详情</DialogTitle>
            <DialogDescription>查看专业素材的基础信息和使用状态。</DialogDescription>
          </DialogHeader>

          {materialViewTarget &&
            (() => {
              const materialDisplay = getReportMaterialDisplayDetails(materialViewTarget)

              return (
                <dl className="mt-4 grid gap-3 text-sm sm:grid-cols-[6rem_minmax(0,1fr)]">
                  <dt className="text-muted-foreground">素材名称</dt>
                  <dd className="min-w-0 font-medium">{materialDisplay.materialName}</dd>
                  <dt className="text-muted-foreground">状态</dt>
                  <dd>{materialDisplay.enabledText}</dd>
                  <dt className="text-muted-foreground">文件名</dt>
                  <dd className="min-w-0 break-all">{materialDisplay.filename}</dd>
                  <dt className="text-muted-foreground">分类</dt>
                  <dd>{materialDisplay.category}</dd>
                  <dt className="text-muted-foreground">类型</dt>
                  <dd>{materialDisplay.materialType}</dd>
                  <dt className="text-muted-foreground">标签</dt>
                  <dd>{materialDisplay.tags}</dd>
                  <dt className="text-muted-foreground">创建时间</dt>
                  <dd>{materialDisplay.createdAt}</dd>
                  <dt className="text-muted-foreground">描述</dt>
                  <dd className="min-w-0 whitespace-pre-wrap">{materialDisplay.description}</dd>
                </dl>
              )
            })()}

          <DialogFooter>
            <Button variant="outline" onClick={() => setMaterialViewTarget(null)}>
              关闭
            </Button>
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

      {/* Delete material confirmation dialog */}
      <Dialog
        open={Boolean(materialDeleteTarget)}
        onOpenChange={(open) => {
          if (!open) {
            setMaterialDeleteTarget(null)
            setMaterialDeleteError(null)
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确定删除此素材？</DialogTitle>
            <DialogDescription>
              <span>
                {materialDeleteTarget?.materialName
                  ? `即将删除素材"${getReportMaterialDisplayDetails(materialDeleteTarget).materialName}"。此操作不可撤销。`
                  : '此操作不可撤销。'}
              </span>
              {materialDeleteError && (
                <span className="mt-2 block text-destructive">{materialDeleteError}</span>
              )}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setMaterialDeleteTarget(null)
                setMaterialDeleteError(null)
              }}
              disabled={deleteMaterialMutation.isPending}
            >
              取消
            </Button>
            <Button
              variant="destructive"
              onClick={handleMaterialDelete}
              disabled={deleteMaterialMutation.isPending}
            >
              {deleteMaterialMutation.isPending ? '删除中...' : '确认删除'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
