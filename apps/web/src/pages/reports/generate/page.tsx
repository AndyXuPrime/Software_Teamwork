import {
  Download,
  FileText,
  GripVertical,
  Loader2,
  Minus,
  PencilLine,
  Play,
  Plus,
  RefreshCw,
  Rocket,
  RotateCcw,
  Save,
  Settings2,
  XCircle,
} from 'lucide-react'
import { type DragEvent, type FormEvent, useCallback, useEffect, useMemo, useState } from 'react'

import { InlineNotice, ProgressSummary, StateBlock } from '@/components/common'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectItemText,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useModelProfiles } from '@/features/admin-config'
import { useCurrentQALLMConfigQuery } from '@/features/qa-settings/qa-settings.queries'
import type {
  CreateReportFormValues,
  Report,
  ReportFile,
  ReportJob,
  ReportJobStatus,
  ReportOutline,
  ReportOutlineNode,
  ReportSection,
  ReportSectionVersion,
} from '@/features/reports'
import {
  createReportSchema,
  formatReportGatewayError,
  getCreateReportDefaults,
  isReportTypeDraftDefaultValue,
  useCancelReportJob,
  useCreateReportFileMutation,
  useCreateReportJobMutation,
  useCreateReportMutation,
  useDownloadReportFileMutation,
  useReportBootstrapQueries,
  useReportDetailQueries,
  useReportJobQuery,
  useReportSettingsQuery,
  useRetryReportJobMutation,
  useSectionVersions,
  useUpdateReportOutlineMutation,
  useUpdateReportSectionMutation,
  useUpdateReportSettingsMutation,
} from '@/features/reports'
import { canAccess } from '@/lib/permissions'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

type FlattenedOutlineNode = {
  node: ReportOutlineNode
  path: number[]
}

type OutlineSectionMetadata = {
  node: ReportOutlineNode
  parentNodeId: string
  sortOrder: number
}

type OutlineEditorState = {
  future: ReportOutlineNode[][]
  nodes: ReportOutlineNode[]
  past: ReportOutlineNode[][]
  sourceKey: string
}

type SectionGenerationReset = {
  attemptCreatedAtMs: number
  jobId: string
  requireFreshSectionTimestamp?: boolean
  reportId: string
  status: Extract<ReportJobStatus, 'pending' | 'running'>
}

const maxOutlineUndoSteps = 15

function cloneOutline(nodes: ReportOutlineNode[]): ReportOutlineNode[] {
  return nodes.map((node) => ({
    ...node,
    children: node.children ? cloneOutline(node.children) : undefined,
  }))
}

function flattenOutline(
  nodes: ReportOutlineNode[],
  parentPath: number[] = [],
): FlattenedOutlineNode[] {
  return nodes.flatMap((node, index) => {
    const path = [...parentPath, index]
    return [{ node, path }, ...flattenOutline(node.children ?? [], path)]
  })
}

function collectOutlineNodeIds(
  nodes: ReportOutlineNode[],
  result = new Set<string>(),
): Set<string> {
  nodes.forEach((node) => {
    if (node.id) result.add(node.id)
    if (node.children) collectOutlineNodeIds(node.children, result)
  })
  return result
}

function flattenOutlineSectionMetadata(
  nodes: ReportOutlineNode[],
  parentNodeId = '',
  result: OutlineSectionMetadata[] = [],
): OutlineSectionMetadata[] {
  nodes.forEach((node) => {
    result.push({ node, parentNodeId, sortOrder: result.length })
    if (node.children) flattenOutlineSectionMetadata(node.children, node.id ?? '', result)
  })
  return result
}

function mergeSectionsWithOutline(
  sections: ReportSection[],
  outline?: ReportOutline | null,
): ReportSection[] {
  if (!outline) return []

  const outlineNodeIds = collectOutlineNodeIds(outline.sections)
  if (outlineNodeIds.size === 0) return sections

  const sectionsByNodeId = new Map<string, ReportSection>()
  sections.forEach((section) => {
    const outlineNodeId = section.outlineNodeId?.trim()
    if (outlineNodeId && outlineNodeIds.has(outlineNodeId)) {
      sectionsByNodeId.set(outlineNodeId, section)
    }
  })

  const sectionIdByNodeId = new Map<string, string>()
  return flattenOutlineSectionMetadata(outline.sections).flatMap(
    ({ node, parentNodeId, sortOrder }) => {
      const nodeId = node.id?.trim()
      if (!nodeId) return []
      const section = sectionsByNodeId.get(nodeId)
      if (!section) return []

      const parentId = parentNodeId ? sectionIdByNodeId.get(parentNodeId) : undefined
      sectionIdByNodeId.set(nodeId, section.id)
      return [
        {
          ...section,
          level: node.level,
          numbering: node.numbering,
          parentId: parentId ?? section.parentId,
          sortOrder,
          title: node.title,
        },
      ]
    },
  )
}

function renumberOutline(
  nodes: ReportOutlineNode[],
  parentNumbering = '',
  depth = 1,
): ReportOutlineNode[] {
  return nodes.map((node, index) => {
    const numbering = parentNumbering ? `${parentNumbering}.${index + 1}` : String(index + 1)
    return {
      ...node,
      children: node.children ? renumberOutline(node.children, numbering, depth + 1) : undefined,
      level: depth,
      numbering,
    }
  })
}

function serializeOutlineNodes(nodes: ReportOutlineNode[]): string {
  return JSON.stringify(
    nodes.map((node) => ({
      children: serializeOutlineNodes(node.children ?? []),
      clientSectionId: node.clientSectionId ?? '',
      id: node.id ?? '',
      level: node.level,
      numbering: node.numbering ?? '',
      title: node.title,
    })),
  )
}

function updateOutlineNodeTitle(
  nodes: ReportOutlineNode[],
  path: number[],
  title: string,
): ReportOutlineNode[] {
  if (path.length === 0) return nodes
  const index = path[0]
  if (index === undefined) return nodes
  const rest = path.slice(1)
  return nodes.map((node, nodeIndex) => {
    if (nodeIndex !== index) return node
    if (rest.length === 0) return { ...node, title }
    return {
      ...node,
      children: updateOutlineNodeTitle(node.children ?? [], rest, title),
    }
  })
}

function createOutlineNode(level: number): ReportOutlineNode {
  return {
    clientSectionId: `new-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    children: [],
    level,
    title: '新章节',
  }
}

function insertOutlineSibling(nodes: ReportOutlineNode[], path: number[]): ReportOutlineNode[] {
  if (path.length === 0) return nodes
  const index = path[0]
  if (index === undefined) return nodes
  const rest = path.slice(1)
  if (rest.length === 0) {
    const current = nodes[index]
    const next = [...nodes]
    next.splice(index + 1, 0, createOutlineNode(current?.level ?? 1))
    return next
  }
  return nodes.map((node, nodeIndex) =>
    nodeIndex === index
      ? { ...node, children: insertOutlineSibling(node.children ?? [], rest) }
      : node,
  )
}

function deleteOutlineNode(nodes: ReportOutlineNode[], path: number[]): ReportOutlineNode[] {
  if (path.length === 0) return nodes
  const index = path[0]
  if (index === undefined) return nodes
  const rest = path.slice(1)
  if (rest.length === 0) {
    return nodes.filter((_, nodeIndex) => nodeIndex !== index)
  }
  return nodes.map((node, nodeIndex) =>
    nodeIndex === index
      ? { ...node, children: deleteOutlineNode(node.children ?? [], rest) }
      : node,
  )
}

function pathsEqual(left: number[] | null, right: number[] | null): boolean {
  if (!left || !right || left.length !== right.length) return false
  return left.every((value, index) => value === right[index])
}

function canMoveOutlineNode(fromPath: number[] | null, toPath: number[]): fromPath is number[] {
  if (!fromPath || fromPath.length !== toPath.length || pathsEqual(fromPath, toPath)) return false
  return fromPath.slice(0, -1).every((value, index) => value === toPath[index])
}

function updateOutlineSiblings(
  nodes: ReportOutlineNode[],
  parentPath: number[],
  updater: (siblings: ReportOutlineNode[]) => ReportOutlineNode[],
): ReportOutlineNode[] {
  if (parentPath.length === 0) return updater(nodes)

  const index = parentPath[0]
  if (index === undefined) return nodes
  const rest = parentPath.slice(1)

  return nodes.map((node, nodeIndex) =>
    nodeIndex === index
      ? { ...node, children: updateOutlineSiblings(node.children ?? [], rest, updater) }
      : node,
  )
}

function moveOutlineNode(
  nodes: ReportOutlineNode[],
  fromPath: number[],
  toPath: number[],
): ReportOutlineNode[] {
  if (!canMoveOutlineNode(fromPath, toPath)) return nodes

  const parentPath = fromPath.slice(0, -1)
  const fromIndex = fromPath[fromPath.length - 1]
  const toIndex = toPath[toPath.length - 1]
  if (fromIndex === undefined || toIndex === undefined) return nodes

  return updateOutlineSiblings(nodes, parentPath, (siblings) => {
    if (
      fromIndex < 0 ||
      fromIndex >= siblings.length ||
      toIndex < 0 ||
      toIndex >= siblings.length
    ) {
      return siblings
    }

    const next = [...siblings]
    const [moved] = next.splice(fromIndex, 1)
    if (!moved) return siblings
    next.splice(Math.min(toIndex, next.length), 0, moved)
    return next
  })
}

const steps = [
  { key: 'draft', label: '1. 草稿与大纲' },
  { key: 'outline', label: '2. 编辑大纲' },
  { key: 'content', label: '3. 正文生成' },
  { key: 'export', label: '4. DOCX 导出' },
] as const

type StepKey = (typeof steps)[number]['key']

const statusText: Record<ReportJobStatus, string> = {
  pending: '等待中',
  running: '生成中',
  succeeded: '已完成',
  partial_succeeded: '部分成功',
  failed: '失败',
  canceled: '已取消',
}

const terminalReportJobStatuses = new Set<ReportJobStatus>([
  'succeeded',
  'partial_succeeded',
  'failed',
  'canceled',
])

function isTerminalReportJobStatus(status: ReportJobStatus): boolean {
  return terminalReportJobStatuses.has(status)
}

function parseTimestampMs(value?: string): number | null {
  if (!value) return null
  const timestamp = Date.parse(value)
  return Number.isFinite(timestamp) ? timestamp : null
}

function sectionUpdatedAfterResetAttempt(
  section: ReportSection,
  reset: SectionGenerationReset,
): boolean {
  const sectionTimestampMs =
    parseTimestampMs(section.generatedAt) ?? parseTimestampMs(section.updatedAt)
  return sectionTimestampMs !== null && sectionTimestampMs >= reset.attemptCreatedAtMs
}

function toActiveGenerationStatus(
  status: ReportJobStatus,
): Extract<ReportJobStatus, 'pending' | 'running'> {
  return status === 'running' ? 'running' : 'pending'
}

type ReportGenerateSession = {
  activeJobId: string | null
  activeSectionId: string
  currentReport: Report | null
  form: CreateReportFormValues
  lastJob: ReportJob | null
  latestFile: ReportFile | null
  selectedMaterialIds: string[]
  step: StepKey
  version: 1
}

const reportGenerateSessionStorageKey = 'report-generation.session.v1'

function createInitialReportForm(): CreateReportFormValues {
  return {
    ...getCreateReportDefaults(''),
    reportType: '',
    templateId: '',
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function isStepKey(value: unknown): value is StepKey {
  return typeof value === 'string' && steps.some((step) => step.key === value)
}

function isReportJobStatus(value: unknown): value is ReportJobStatus {
  return typeof value === 'string' && value in statusText
}

function isReportSnapshot(value: unknown): value is Report {
  return (
    isRecord(value) &&
    typeof value.id === 'string' &&
    typeof value.name === 'string' &&
    typeof value.reportType === 'string'
  )
}

function isReportJobSnapshot(value: unknown): value is ReportJob {
  return (
    isRecord(value) &&
    typeof value.id === 'string' &&
    typeof value.reportId === 'string' &&
    isReportJobStatus(value.status) &&
    typeof value.jobType === 'string'
  )
}

function isReportFileSnapshot(value: unknown): value is ReportFile {
  return isRecord(value) && typeof value.id === 'string'
}

function inferRestoredStep(step: unknown, job: ReportJob | null): StepKey {
  if (isStepKey(step)) return step
  if (isContentJob(job)) return 'content'
  if (isOutlineJob(job)) return 'outline'
  return 'draft'
}

function reportGenerationStorage(): Storage | null {
  if (typeof window === 'undefined') return null
  try {
    return window.sessionStorage
  } catch {
    return null
  }
}

function readReportGenerateSession(): ReportGenerateSession | null {
  const storage = reportGenerationStorage()
  if (!storage) return null
  const raw = storage.getItem(reportGenerateSessionStorageKey)
  if (!raw) return null

  try {
    const parsed: unknown = JSON.parse(raw)
    if (!isRecord(parsed)) return null

    const currentReport = isReportSnapshot(parsed.currentReport) ? parsed.currentReport : null
    const lastJob = isReportJobSnapshot(parsed.lastJob) ? parsed.lastJob : null
    const latestFile = isReportFileSnapshot(parsed.latestFile) ? parsed.latestFile : null
    const activeJobId =
      typeof parsed.activeJobId === 'string' && parsed.activeJobId.trim() !== ''
        ? parsed.activeJobId
        : (lastJob?.id ?? null)
    const selectedMaterialIds = Array.isArray(parsed.selectedMaterialIds)
      ? parsed.selectedMaterialIds.filter((item): item is string => typeof item === 'string')
      : []
    const formResult = createReportSchema.safeParse(parsed.form)

    if (!currentReport && !lastJob && !activeJobId && !latestFile) return null

    return {
      activeJobId,
      activeSectionId: typeof parsed.activeSectionId === 'string' ? parsed.activeSectionId : '',
      currentReport,
      form: formResult.success ? formResult.data : createInitialReportForm(),
      lastJob,
      latestFile,
      selectedMaterialIds,
      step: inferRestoredStep(parsed.step, lastJob),
      version: 1,
    }
  } catch {
    storage.removeItem(reportGenerateSessionStorageKey)
    return null
  }
}

function writeReportGenerateSession(session: ReportGenerateSession | null) {
  const storage = reportGenerationStorage()
  if (!storage) return
  if (!session) {
    storage.removeItem(reportGenerateSessionStorageKey)
    return
  }
  storage.setItem(reportGenerateSessionStorageKey, JSON.stringify(session))
}

const outlineProgressMax = 20
const outlineRunningProgressCap = outlineProgressMax - 2
const outlineRunningProgressMin = 5
const outlineRunningSmoothDurationMs = 90 * 1000
const activeProgressTickMs = 800

function clampProgress(value: number): number {
  if (!Number.isFinite(value)) return 0
  return Math.max(0, Math.min(100, Math.round(value)))
}

function readProgressNumber(
  progress: ReportJob['progress'] | undefined,
  keys: string[],
): number | null {
  if (!progress || typeof progress !== 'object') return null
  for (const key of keys) {
    const value = progress[key]
    if (typeof value === 'number' && Number.isFinite(value)) return value
    if (typeof value === 'string' && value.trim() !== '') {
      const parsed = Number(value)
      if (Number.isFinite(parsed)) return parsed
    }
  }
  return null
}

function getProgressRatio(job?: ReportJob | null): number | null {
  const completed = readProgressNumber(job?.progress, ['completed', 'completedSections'])
  const total = readProgressNumber(job?.progress, ['total', 'totalSections'])
  if (completed === null || total === null || total <= 0) return null
  return Math.max(0, Math.min(1, completed / total))
}

function hashProgressSeed(value: string): number {
  let hash = 0
  for (let index = 0; index < value.length; index += 1) {
    hash = (hash * 31 + value.charCodeAt(index)) % 1009
  }
  return hash
}

function getOutlineProgressJitter(jobId: string, elapsedMs: number): number {
  const bucket = Math.floor(elapsedMs / 7000)
  const seed = hashProgressSeed(`${jobId}:${bucket}`)
  return (seed / 1009 - 0.5) * 1.2
}

function getOutlineRunningProgress(job: ReportJob, now = Date.now()): number {
  const startedAt = Date.parse(job.startedAt ?? job.createdAt)
  if (!Number.isFinite(startedAt)) return outlineRunningProgressMin
  const elapsedMs = Math.max(0, now - startedAt)
  const ratio = Math.min(1, elapsedMs / outlineRunningSmoothDurationMs)
  const base =
    outlineRunningProgressMin + ratio * (outlineRunningProgressCap - outlineRunningProgressMin)
  const jitter = ratio < 0.08 ? 0 : getOutlineProgressJitter(job.id, elapsedMs)
  return Math.min(outlineRunningProgressCap, Math.max(outlineRunningProgressMin, base + jitter))
}

function isContentJob(job?: ReportJob | null): boolean {
  return (
    job?.jobType === 'content_generation' ||
    job?.jobType === 'content_regeneration' ||
    job?.jobType === 'section_regeneration'
  )
}

function isOutlineJob(job?: ReportJob | null): boolean {
  return job?.jobType === 'outline_generation' || job?.jobType === 'outline_regeneration'
}

function isResetAttemptFinished(job: ReportJob, reset: SectionGenerationReset): boolean {
  if (job.id !== reset.jobId) return false
  if (!isTerminalReportJobStatus(job.status)) return false
  const finishedAtMs = parseTimestampMs(job.finishedAt)
  return finishedAtMs !== null && finishedAtMs >= reset.attemptCreatedAtMs
}

function shouldApplySectionGenerationReset(
  reset: SectionGenerationReset | null,
  job?: ReportJob | null,
): reset is SectionGenerationReset {
  if (!reset || !job || job.id !== reset.jobId || job.reportId !== reset.reportId) return false
  return !isResetAttemptFinished(job, reset)
}

function getRetryAwareJob(
  job: ReportJob | null | undefined,
  reset: SectionGenerationReset | null,
): ReportJob | null | undefined {
  if (!reset || !job || !shouldApplySectionGenerationReset(reset, job)) return job
  if (job.status === 'pending' || job.status === 'running') return job

  return {
    ...job,
    error: undefined,
    finishedAt: undefined,
    progress: {},
    resultSummary: undefined,
    startedAt: undefined,
    status: reset.status,
  }
}

function getRetryAwareSections(
  sections: ReportSection[],
  reset: SectionGenerationReset | null,
  job?: ReportJob | null,
): ReportSection[] {
  if (!shouldApplySectionGenerationReset(reset, job) || !isContentJob(job)) return sections

  return sections.map((section) => {
    if (section.lastJobId === reset.jobId) {
      if (!reset.requireFreshSectionTimestamp || sectionUpdatedAfterResetAttempt(section, reset)) {
        return section
      }
    }

    if (section.generationStatus === 'pending') {
      return section
    }

    return {
      ...section,
      generatedAt: undefined,
      generationStatus: reset.status,
      lastJobId: undefined,
    }
  })
}

function getProgressPercent(job?: ReportJob | null, now = Date.now()): number {
  if (!job) return 0

  const ratio = getProgressRatio(job)
  const explicitPercent = readProgressNumber(job.progress, ['percent'])
  const fallbackPercent = explicitPercent === null ? null : clampProgress(explicitPercent)

  if (isOutlineJob(job)) {
    if (job.status === 'succeeded' || job.status === 'partial_succeeded') return outlineProgressMax
    const ratioProgress = ratio === null ? 0 : ratio * outlineProgressMax
    if (job.status === 'pending' || job.status === 'running') {
      const smoothProgress = getOutlineRunningProgress(job, now)
      return clampProgress(
        Math.min(outlineRunningProgressCap, Math.max(ratioProgress, smoothProgress)),
      )
    }
    if (ratio === null && fallbackPercent !== null) return fallbackPercent
    return clampProgress(ratioProgress)
  }

  if (isContentJob(job)) {
    if (ratio === null && fallbackPercent !== null) return fallbackPercent
    if (job.status === 'succeeded' || job.status === 'partial_succeeded') return 100
    const ratioProgress =
      ratio === null ? 0 : outlineProgressMax + ratio * (100 - outlineProgressMax)
    if (job.status === 'pending' || job.status === 'running') {
      return clampProgress(Math.max(outlineProgressMax, ratioProgress))
    }
    return clampProgress(ratioProgress)
  }

  if (ratio === null && fallbackPercent !== null) return fallbackPercent
  if (job.status === 'succeeded' || job.status === 'partial_succeeded') return 100
  return clampProgress(ratio === null ? 0 : ratio * 100)
}

function getJobProgressLabel(job?: ReportJob | null): string {
  if (!job) return '-'
  if (isOutlineJob(job) && (job.status === 'succeeded' || job.status === 'partial_succeeded')) {
    return '大纲生成完成'
  }
  if (isContentJob(job) && job.status === 'succeeded') return '报告生成完成'
  if (isContentJob(job) && job.status === 'partial_succeeded') return '章节处理完成，部分失败'
  return statusText[job.status]
}

function formatDate(value?: string): string {
  if (!value) return '-'
  return new Date(value).toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function shouldApplyReportTypeDefault(
  field: 'businessObject' | 'extraContextText' | 'name' | 'specialty' | 'topic',
  form: CreateReportFormValues,
  force: boolean,
): boolean {
  const value = form[field]
  return force || !value || isReportTypeDraftDefaultValue(field, value)
}

function applyReportTypeDraftDefaults(
  form: CreateReportFormValues,
  reportType: string,
  options: { force?: boolean } = {},
): CreateReportFormValues {
  const force = options.force ?? false
  const defaults = getCreateReportDefaults(reportType)

  return {
    ...form,
    reportType,
    businessObject: shouldApplyReportTypeDefault('businessObject', form, force)
      ? defaults.businessObject
      : form.businessObject,
    extraContextText: shouldApplyReportTypeDefault('extraContextText', form, force)
      ? defaults.extraContextText
      : form.extraContextText,
    name: shouldApplyReportTypeDefault('name', form, force) ? defaults.name : form.name,
    specialty: shouldApplyReportTypeDefault('specialty', form, force)
      ? defaults.specialty
      : form.specialty,
    topic: shouldApplyReportTypeDefault('topic', form, force) ? defaults.topic : form.topic,
  }
}

export function ReportGeneratePage() {
  const [restoredSession] = useState<ReportGenerateSession | null>(() =>
    readReportGenerateSession(),
  )
  const [step, setStep] = useState<StepKey>(() => restoredSession?.step ?? 'draft')
  const [form, setForm] = useState<CreateReportFormValues>(
    () => restoredSession?.form ?? createInitialReportForm(),
  )
  const [selectedMaterialIds, setSelectedMaterialIds] = useState<string[]>(
    () => restoredSession?.selectedMaterialIds ?? [],
  )
  const [currentReport, setCurrentReport] = useState<Report | null>(
    () => restoredSession?.currentReport ?? null,
  )
  const [activeJobId, setActiveJobId] = useState<string | null>(
    () => restoredSession?.activeJobId ?? restoredSession?.lastJob?.id ?? null,
  )
  const [lastJob, setLastJob] = useState<ReportJob | null>(() => restoredSession?.lastJob ?? null)
  const [latestFile, setLatestFile] = useState<ReportFile | null>(
    () => restoredSession?.latestFile ?? null,
  )
  const [activeSectionId, setActiveSectionId] = useState(
    () => restoredSession?.activeSectionId ?? '',
  )
  const [sectionDraft, setSectionDraft] = useState('')
  const [showVersions, setShowVersions] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)
  const [formError, setFormError] = useState<string | null>(null)
  const [documentProfileId, setDocumentProfileId] = useState('')
  const [documentProfileTouched, setDocumentProfileTouched] = useState(false)
  const [documentSettingsNotice, setDocumentSettingsNotice] = useState<string | null>(null)
  const [outlineEditor, setOutlineEditor] = useState<OutlineEditorState>({
    future: [],
    nodes: [],
    past: [],
    sourceKey: '',
  })
  const [draggedOutlinePath, setDraggedOutlinePath] = useState<number[] | null>(null)
  const [dragOverOutlinePath, setDragOverOutlinePath] = useState<number[] | null>(null)
  const [progressNow, setProgressNow] = useState(() => Date.now())
  const [sectionGenerationReset, setSectionGenerationReset] =
    useState<SectionGenerationReset | null>(null)

  const user = useAuthStore((state) => state.user)
  const canManageDocumentModelSettings =
    canAccess(user, { any: ['admin:model-profile:write', 'system:admin'] }) ||
    canAccess(user, { roles: ['system:admin'] })
  const { typeQuery, templateQuery, materialQuery } = useReportBootstrapQueries(form.reportType)
  const reportSettingsQuery = useReportSettingsQuery({
    enabled: canManageDocumentModelSettings,
  })
  const userDocumentModelQuery = useCurrentQALLMConfigQuery({
    enabled: Boolean(user) && !canManageDocumentModelSettings,
  })
  const chatProfilesQuery = useModelProfiles('chat', true, {
    queryEnabled: canManageDocumentModelSettings,
  })
  const jobQuery = useReportJobQuery(activeJobId)
  const activeJobForPolling = jobQuery.data ?? lastJob
  const { outlinesQuery, sectionsQuery } = useReportDetailQueries(
    currentReport?.id ?? null,
    activeJobForPolling,
  )
  const createReportMutation = useCreateReportMutation()
  const createJobMutation = useCreateReportJobMutation()
  const saveOutlineMutation = useUpdateReportOutlineMutation(currentReport?.id ?? '')
  const saveSectionMutation = useUpdateReportSectionMutation(currentReport?.id ?? '')
  const createFileMutation = useCreateReportFileMutation()
  const updateReportSettingsMutation = useUpdateReportSettingsMutation()
  const retryJobMutation = useRetryReportJobMutation()
  const cancelJobMutation = useCancelReportJob()
  const downloadMutation = useDownloadReportFileMutation()
  const sectionVersionsQuery = useSectionVersions(
    currentReport?.id ?? null,
    showVersions ? activeSectionId : null,
  )

  const reportTypes = useMemo(() => typeQuery.data ?? [], [typeQuery.data])
  const templates = useMemo(() => templateQuery.data?.items ?? [], [templateQuery.data])
  const materials = useMemo(() => materialQuery.data?.items ?? [], [materialQuery.data])
  const chatProfiles = useMemo(() => chatProfilesQuery.data ?? [], [chatProfilesQuery.data])
  const currentOutline = outlinesQuery.data?.[0]
  const outlineSourceKey = currentOutline
    ? `${currentOutline.id}:${currentOutline.version}:${currentOutline.sections.length}`
    : ''
  const outline = outlineEditor.nodes
  const numberedOutline = useMemo(() => renumberOutline(outline), [outline])
  const savedNumberedOutline = useMemo(
    () => renumberOutline(currentOutline?.sections ?? []),
    [currentOutline?.sections],
  )
  const isOutlineDirty = useMemo(
    () =>
      Boolean(currentOutline) &&
      serializeOutlineNodes(numberedOutline) !== serializeOutlineNodes(savedNumberedOutline),
    [currentOutline, numberedOutline, savedNumberedOutline],
  )
  const flattenedOutline = useMemo(() => flattenOutline(numberedOutline), [numberedOutline])
  const effectiveJob = useMemo(
    () => getRetryAwareJob(activeJobForPolling, sectionGenerationReset),
    [activeJobForPolling, sectionGenerationReset],
  )
  const currentOutlineSections = useMemo(
    () => mergeSectionsWithOutline(sectionsQuery.data ?? [], currentOutline),
    [currentOutline, sectionsQuery.data],
  )
  const sections = useMemo(
    () => getRetryAwareSections(currentOutlineSections, sectionGenerationReset, effectiveJob),
    [currentOutlineSections, effectiveJob, sectionGenerationReset],
  )
  const activeSection = sections.find((item) => item.id === activeSectionId) ?? sections[0]
  const selectedTemplate = templates.find((template) => template.id === form.templateId)
  const selectedReportType = reportTypes.find(
    (type) => type.code === (currentReport?.reportType ?? form.reportType),
  )
  const reportTemplateTypeLabel =
    selectedReportType?.name ??
    selectedTemplate?.reportType ??
    currentReport?.reportType ??
    form.reportType ??
    '-'
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
  const hasDraftPendingOutlineJob = Boolean(currentReport && step === 'draft')

  const bootstrapErrors = useMemo(
    () =>
      [
        { error: typeQuery.error, label: '报告类型', visible: typeQuery.isError },
        { error: templateQuery.error, label: '报告模板', visible: templateQuery.isError },
        { error: materialQuery.error, label: '报告素材', visible: materialQuery.isError },
      ].filter((item) => item.visible),
    [
      materialQuery.error,
      materialQuery.isError,
      templateQuery.error,
      templateQuery.isError,
      typeQuery.error,
      typeQuery.isError,
    ],
  )
  const isBootstrapLoading =
    typeQuery.isLoading || templateQuery.isLoading || materialQuery.isLoading
  const canCreateReport =
    !isBootstrapLoading &&
    !typeQuery.isError &&
    !templateQuery.isError &&
    reportTypes.length > 0 &&
    templates.length > 0

  useEffect(() => {
    if (reportTypes.length === 0) return
    if (reportTypes.some((type) => type.code === form.reportType)) return
    setForm((prev) =>
      applyReportTypeDraftDefaults(prev, reportTypes[0]?.code ?? '', {
        force: !prev.reportType,
      }),
    )
  }, [form.reportType, reportTypes])

  useEffect(() => {
    const firstTemplate = templates[0]
    const hasSelectedTemplate = templates.some((template) => template.id === form.templateId)
    if (firstTemplate && !hasSelectedTemplate) {
      setForm((prev) => ({ ...prev, templateId: firstTemplate.id }))
    } else if (!firstTemplate && form.templateId) {
      setForm((prev) => ({ ...prev, templateId: '' }))
    }
  }, [form.templateId, templates])

  useEffect(() => {
    if (!canManageDocumentModelSettings) {
      setDocumentProfileId('')
      setDocumentProfileTouched(false)
      setDocumentSettingsNotice(null)
      return
    }

    if (documentProfileTouched) return
    if (!reportSettingsQuery.isSuccess) {
      setDocumentProfileId('')
      return
    }

    const nextProfileId = configuredDocumentProfileId || firstChatProfileId
    setDocumentProfileId(nextProfileId)
  }, [
    canManageDocumentModelSettings,
    configuredDocumentProfileId,
    documentProfileTouched,
    firstChatProfileId,
    reportSettingsQuery.isSuccess,
  ])

  useEffect(() => {
    if (sections.length === 0) {
      if (activeSectionId) setActiveSectionId('')
      return
    }
    if (!sections.some((section) => section.id === activeSectionId)) {
      setActiveSectionId(sections[0]?.id ?? '')
    }
  }, [activeSectionId, sections])

  useEffect(() => {
    if (activeSection) {
      setSectionDraft(activeSection.content ?? '')
    } else {
      setSectionDraft('')
    }
  }, [activeSection])

  useEffect(() => {
    if (jobQuery.data) {
      setLastJob(jobQuery.data)
    }
  }, [jobQuery.data])

  useEffect(() => {
    if (!sectionGenerationReset || !activeJobForPolling) return
    if (isResetAttemptFinished(activeJobForPolling, sectionGenerationReset)) {
      setSectionGenerationReset(null)
    }
  }, [activeJobForPolling, sectionGenerationReset])

  useEffect(() => {
    if (effectiveJob?.status !== 'pending' && effectiveJob?.status !== 'running') return

    setProgressNow(Date.now())
    const intervalId = window.setInterval(() => {
      setProgressNow(Date.now())
    }, activeProgressTickMs)

    return () => window.clearInterval(intervalId)
  }, [effectiveJob?.id, effectiveJob?.status])

  useEffect(() => {
    const hasGenerationSession = Boolean(currentReport || activeJobId || lastJob || latestFile)
    if (!hasGenerationSession) {
      writeReportGenerateSession(null)
      return
    }

    writeReportGenerateSession({
      activeJobId,
      activeSectionId,
      currentReport,
      form,
      lastJob,
      latestFile,
      selectedMaterialIds,
      step,
      version: 1,
    })
  }, [
    activeJobId,
    activeSectionId,
    currentReport,
    form,
    lastJob,
    latestFile,
    selectedMaterialIds,
    step,
  ])

  useEffect(() => {
    setOutlineEditor((prev) => {
      if (!currentOutline) {
        return prev.sourceKey === '' ? prev : { future: [], nodes: [], past: [], sourceKey: '' }
      }
      if (prev.sourceKey === outlineSourceKey) return prev
      return {
        future: [],
        nodes: cloneOutline(currentOutline.sections),
        past: [],
        sourceKey: outlineSourceKey,
      }
    })
  }, [currentOutline, outlineSourceKey])

  const commitOutlineChange = (updater: (nodes: ReportOutlineNode[]) => ReportOutlineNode[]) => {
    setOutlineEditor((prev) => {
      const previousNodes = cloneOutline(prev.nodes)
      const nextNodes = updater(cloneOutline(prev.nodes))
      return {
        ...prev,
        future: [],
        nodes: nextNodes,
        past: [...prev.past, previousNodes].slice(-maxOutlineUndoSteps),
      }
    })
  }

  const handleOutlineTitleChange = (path: number[], title: string) => {
    commitOutlineChange((nodes) => updateOutlineNodeTitle(nodes, path, title))
  }

  const handleAddOutlineSibling = (path: number[]) => {
    commitOutlineChange((nodes) => insertOutlineSibling(nodes, path))
  }

  const handleDeleteOutlineNode = (path: number[]) => {
    commitOutlineChange((nodes) => deleteOutlineNode(nodes, path))
  }

  const handleOutlineDragStart = (event: DragEvent<HTMLElement>, path: number[]) => {
    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData('text/plain', path.join('.'))
    setDraggedOutlinePath(path)
  }

  const handleOutlineDragOver = (event: DragEvent<HTMLElement>, path: number[]) => {
    if (!canMoveOutlineNode(draggedOutlinePath, path)) return
    event.preventDefault()
    event.dataTransfer.dropEffect = 'move'
    setDragOverOutlinePath(path)
  }

  const handleOutlineDrop = (event: DragEvent<HTMLElement>, path: number[]) => {
    if (!canMoveOutlineNode(draggedOutlinePath, path)) return
    event.preventDefault()
    commitOutlineChange((nodes) => moveOutlineNode(nodes, draggedOutlinePath, path))
    setDraggedOutlinePath(null)
    setDragOverOutlinePath(null)
  }

  const handleOutlineDragEnd = () => {
    setDraggedOutlinePath(null)
    setDragOverOutlinePath(null)
  }

  const undoOutline = useCallback(() => {
    setOutlineEditor((prev) => {
      const previous = prev.past[prev.past.length - 1]
      if (!previous) return prev
      return {
        ...prev,
        future: [cloneOutline(prev.nodes), ...prev.future].slice(0, maxOutlineUndoSteps),
        nodes: cloneOutline(previous),
        past: prev.past.slice(0, -1),
      }
    })
  }, [])

  const redoOutline = useCallback(() => {
    setOutlineEditor((prev) => {
      const next = prev.future[0]
      if (!next) return prev
      return {
        ...prev,
        future: prev.future.slice(1),
        nodes: cloneOutline(next),
        past: [...prev.past, cloneOutline(prev.nodes)].slice(-maxOutlineUndoSteps),
      }
    })
  }, [])

  useEffect(() => {
    if (step !== 'outline') return

    const handleOutlineKeyDown = (event: KeyboardEvent) => {
      if (!event.ctrlKey && !event.metaKey) return
      const key = event.key.toLowerCase()
      if (key === 'z' && !event.shiftKey) {
        event.preventDefault()
        undoOutline()
        return
      }
      if (key === 'y' || (key === 'z' && event.shiftKey)) {
        event.preventDefault()
        redoOutline()
      }
    }

    document.addEventListener('keydown', handleOutlineKeyDown)
    return () => document.removeEventListener('keydown', handleOutlineKeyDown)
  }, [redoOutline, step, undoOutline])

  const updateForm = (field: keyof CreateReportFormValues, value: string | number) => {
    setForm((prev) => ({ ...prev, [field]: value }))
  }

  const handleSelectDocumentProfile = (profileId: string) => {
    setDocumentProfileTouched(true)
    setDocumentProfileId(profileId)
    setDocumentSettingsNotice(null)
  }

  const handlePublishDocumentProfile = async () => {
    const profileId = documentProfileId.trim()
    setDocumentSettingsNotice(null)

    if (!canManageDocumentModelSettings) {
      setDocumentSettingsNotice('当前账号无权发布文档生成模型配置。')
      return
    }
    if (reportSettingsQuery.isLoading) {
      setDocumentSettingsNotice('正在读取当前文档生成模型配置，请稍后再发布。')
      return
    }
    if (!profileId) {
      setDocumentSettingsNotice('请选择用于报告生成的文档生成模型。')
      return
    }

    try {
      await updateReportSettingsMutation.mutateAsync({
        llm: { profileId, provider: 'ai-gateway' },
      })
      setDocumentSettingsNotice('文档生成模型配置已发布。')
    } catch (error) {
      setDocumentSettingsNotice(formatReportGatewayError(error, '文档生成模型配置发布失败'))
    }
  }

  const toggleMaterial = (id: string) => {
    setSelectedMaterialIds((prev) =>
      prev.includes(id) ? prev.filter((item) => item !== id) : [...prev, id],
    )
  }

  const handleRestartDraft = () => {
    writeReportGenerateSession(null)
    setStep('draft')
    setForm(createInitialReportForm())
    setSelectedMaterialIds([])
    setCurrentReport(null)
    setActiveJobId(null)
    setLastJob(null)
    setLatestFile(null)
    setActiveSectionId('')
    setSectionDraft('')
    setShowVersions(false)
    setNotice(null)
    setFormError(null)
    setOutlineEditor({ future: [], nodes: [], past: [], sourceKey: '' })
    setSectionGenerationReset(null)
    createReportMutation.reset()
    createJobMutation.reset()
    saveOutlineMutation.reset()
    saveSectionMutation.reset()
    createFileMutation.reset()
    retryJobMutation.reset()
    cancelJobMutation.reset()
    downloadMutation.reset()
  }

  const handleCreateReport = async (event: FormEvent) => {
    event.preventDefault()
    setFormError(null)
    setNotice(null)

    const parsed = createReportSchema.safeParse(form)
    if (!parsed.success) {
      setFormError(parsed.error.issues[0]?.message ?? '请检查报告参数')
      return
    }

    const payload = {
      name: parsed.data.name,
      reportType: parsed.data.reportType,
      templateId: parsed.data.templateId,
      topic: parsed.data.topic,
      specialty: parsed.data.specialty,
      businessObject: parsed.data.businessObject,
      year: parsed.data.year,
      source: 'frontend' as const,
    }

    let report = currentReport
    if (!report) {
      try {
        report = await createReportMutation.mutateAsync(payload)
        setCurrentReport(report)
      } catch (error) {
        setActiveJobId(null)
        setNotice(formatReportGatewayError(error, '创建报告草稿失败'))
        return
      }
    }

    setActiveJobId(null)
    setLastJob(null)
    setActiveSectionId('')
    setSectionDraft('')
    setShowVersions(false)
    setLatestFile(null)
    setOutlineEditor({ future: [], nodes: [], past: [], sourceKey: '' })
    setSectionGenerationReset(null)

    try {
      const job = await createJobMutation.mutateAsync({
        reportId: report.id,
        payload: {
          jobType: 'outline_generation',
          target: { scope: 'report' },
          materialIds: selectedMaterialIds,
          requirements: parsed.data.extraContextText,
        },
      })
      setLastJob(job)
      setActiveJobId(job.id)
      setStep('outline')
      setNotice('已创建报告草稿，正在生成大纲。页面会根据服务端返回的大纲和进度自动更新。')
    } catch (error) {
      setActiveJobId(null)
      setNotice(
        `${formatReportGatewayError(
          error,
          '创建大纲任务失败',
        )}；已保留服务端报告草稿"${report.name}"，再次提交将复用该草稿创建大纲任务。`,
      )
    }
  }

  const handleSaveOutline = async () => {
    if (!currentReport || !outlinesQuery.data?.[0]) {
      setNotice('暂无可保存的服务端大纲。请先创建报告并等待大纲接口返回数据。')
      return
    }

    try {
      await saveOutlineMutation.mutateAsync({
        outlineId: outlinesQuery.data[0].id,
        sections: numberedOutline,
      })
      setNotice('大纲已保存，章节编号已按当前顺序重新生成。')
    } catch (error) {
      setNotice(formatReportGatewayError(error, '大纲保存失败'))
    }
  }

  const handleGenerateContent = async () => {
    if (!currentReport) {
      setNotice('请先创建报告草稿。')
      return
    }
    if (!currentOutline) {
      setNotice('暂无可用的服务端大纲，不能创建正文生成任务。')
      return
    }
    if (outline.length === 0) {
      setNotice('暂无服务端大纲数据，不能创建正文生成任务。')
      return
    }

    if (isOutlineDirty) {
      try {
        await saveOutlineMutation.mutateAsync({
          outlineId: currentOutline.id,
          sections: numberedOutline,
        })
      } catch (error) {
        setNotice(formatReportGatewayError(error, '大纲保存失败，未创建正文生成任务'))
        return
      }
    }

    try {
      const job = await createJobMutation.mutateAsync({
        reportId: currentReport.id,
        payload: {
          jobType: 'content_generation',
          target: { scope: 'report' },
          materialIds: selectedMaterialIds,
          options: { preserveManualEdits: true, saveResult: true },
        },
      })
      setLastJob(job)
      setActiveJobId(job.id)
      if (job.status === 'pending' || job.status === 'running') {
        setSectionGenerationReset({
          attemptCreatedAtMs: parseTimestampMs(job.createdAt) ?? Date.now(),
          jobId: job.id,
          reportId: job.reportId,
          status: toActiveGenerationStatus(job.status),
        })
      } else {
        setSectionGenerationReset(null)
      }
      setStep('content')
      setNotice('已开始生成正文。每完成一个章节，进度会继续更新。')
    } catch (error) {
      setNotice(formatReportGatewayError(error, '正文生成任务创建失败'))
    }
  }

  const handleSaveSection = async () => {
    if (!currentReport || !activeSection) {
      setNotice('暂无可保存的服务端章节。')
      return
    }

    try {
      await saveSectionMutation.mutateAsync({
        sectionId: activeSection.id,
        title: activeSection.title,
        content: sectionDraft,
      })
      setNotice('章节正文已保存。')
    } catch (error) {
      setNotice(formatReportGatewayError(error, '章节保存失败'))
    }
  }

  const handleRetry = async () => {
    const retryJob = effectiveJob ?? lastJob
    if (retryJob?.id) {
      try {
        const attempt = await retryJobMutation.mutateAsync({
          jobId: retryJob.id,
          reportId: retryJob.reportId,
        })
        const retryStatus = toActiveGenerationStatus(attempt.status)
        setSectionGenerationReset({
          attemptCreatedAtMs: parseTimestampMs(attempt.createdAt) ?? Date.now(),
          jobId: retryJob.id,
          requireFreshSectionTimestamp: true,
          reportId: retryJob.reportId,
          status: retryStatus,
        })
        setLastJob({
          ...retryJob,
          error: undefined,
          finishedAt: undefined,
          progress: {},
          resultSummary: undefined,
          startedAt: undefined,
          status: retryStatus,
        })
        setActiveJobId(retryJob.id)
        setNotice(`已重新提交任务，当前状态：${statusText[attempt.status]}`)
      } catch (error) {
        setNotice(formatReportGatewayError(error, '创建重试尝试失败'))
      }
      return
    }
    setNotice('暂无可重试的服务端任务。')
  }

  const handleCancelJob = async () => {
    const cancelJob = effectiveJob ?? lastJob
    if (cancelJob?.id) {
      try {
        const job = await cancelJobMutation.mutateAsync(cancelJob.id)
        setLastJob(job)
        setNotice(`已提交取消任务，当前状态：${statusText[job.status]}`)
      } catch (error) {
        setNotice(formatReportGatewayError(error, '取消任务失败'))
      }
      return
    }
    setNotice('暂无可取消的服务端任务。')
  }

  const handleExport = async () => {
    if (!currentReport) {
      setNotice('请先创建报告草稿。')
      return
    }

    try {
      const file = await createFileMutation.mutateAsync({
        reportId: currentReport.id,
        format: 'docx',
        templateId: selectedTemplate?.id,
        styleOptions: { numberingMode: 'global' },
      })
      setLatestFile(file)
      setStep('export')
      setNotice('已创建 DOCX 文件资源，可以在文件就绪后下载。')
    } catch (error) {
      setNotice(formatReportGatewayError(error, '创建 DOCX 文件资源失败'))
    }
  }

  const handleDownload = async () => {
    if (!latestFile) {
      setNotice('暂无可下载的服务端文件资源。')
      return
    }

    try {
      const blob = await downloadMutation.mutateAsync(latestFile.id)
      const url = URL.createObjectURL(blob)
      const anchor = document.createElement('a')
      anchor.href = url
      anchor.download = latestFile.filename ?? `${form.name}.docx`
      anchor.click()
      URL.revokeObjectURL(url)
    } catch (error) {
      setNotice(formatReportGatewayError(error, '下载报告文件失败'))
    }
  }

  const progressPercent = getProgressPercent(effectiveJob, progressNow)
  const jobStatusLabel = getJobProgressLabel(effectiveJob)
  const jobProgressTone =
    effectiveJob?.status === 'failed'
      ? 'error'
      : effectiveJob?.status === 'canceled'
        ? 'warning'
        : effectiveJob?.status === 'succeeded' || effectiveJob?.status === 'partial_succeeded'
          ? 'success'
          : 'default'
  const canRetryJob =
    effectiveJob?.status === 'failed' ||
    effectiveJob?.status === 'succeeded' ||
    effectiveJob?.status === 'partial_succeeded' ||
    effectiveJob?.status === 'canceled'
  const canCancelJob = effectiveJob?.status === 'pending' || effectiveJob?.status === 'running'

  return (
    <div className="flex h-full flex-col overflow-auto bg-background">
      <div className="border-b border-border bg-muted/30 px-6 py-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h1 className="text-xl font-semibold text-foreground">报告生成</h1>
            <p className="mt-1 text-sm text-muted-foreground">
              按最新 gateway RESTful 契约整合：草稿、大纲、正文任务和 DOCX 文件资源。
            </p>
          </div>
          <div className="flex flex-wrap gap-2">
            {steps.map((item) => (
              <Button
                key={item.key}
                type="button"
                variant={step === item.key ? 'default' : 'outline'}
                size="sm"
                onClick={() => setStep(item.key)}
              >
                {item.label}
              </Button>
            ))}
          </div>
        </div>

        {bootstrapErrors.map((item) => (
          <InlineNotice
            className="mt-3"
            key={item.label}
            title={`${item.label}加载失败`}
            variant="error"
          >
            {formatReportGatewayError(item.error, `${item.label}加载失败`)}
          </InlineNotice>
        ))}
        {(notice || formError) && (
          <InlineNotice
            className="mt-4"
            title={formError ? '表单校验失败' : undefined}
            variant={formError ? 'error' : 'info'}
          >
            {formError ?? notice}
          </InlineNotice>
        )}
      </div>

      <div className="grid flex-1 gap-6 p-6 xl:grid-cols-[minmax(0,1.1fr)_360px]">
        <div className="min-w-0 space-y-6">
          {(currentReport || effectiveJob) && (
            <section className="rounded-lg border border-border bg-card p-5">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <h2 className="text-base font-semibold">当前文档进度</h2>
                </div>
                <div className="flex flex-wrap gap-2">
                  {canCancelJob && (
                    <Button
                      variant="outline"
                      size="sm"
                      className="text-foreground"
                      onClick={handleCancelJob}
                      disabled={cancelJobMutation.isPending}
                    >
                      {cancelJobMutation.isPending ? (
                        <Loader2 className="size-3 animate-spin" />
                      ) : (
                        <XCircle className="size-3" />
                      )}
                      取消任务
                    </Button>
                  )}
                  {canRetryJob && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleRetry}
                      disabled={retryJobMutation.isPending}
                    >
                      {retryJobMutation.isPending ? (
                        <Loader2 className="size-3 animate-spin" />
                      ) : (
                        <RefreshCw className="size-3" />
                      )}
                      重试任务
                    </Button>
                  )}
                </div>
              </div>

              <div className="mt-4 grid gap-3 text-sm md:grid-cols-2">
                <div className="flex justify-between gap-4 rounded-lg border border-border bg-background px-3 py-2">
                  <span className="text-muted-foreground">状态</span>
                  <span
                    className={cn(
                      effectiveJob?.status === 'failed' && 'text-destructive',
                      effectiveJob?.status === 'canceled' && 'text-yellow-600',
                      effectiveJob?.status === 'succeeded' && 'text-green-600',
                      (effectiveJob?.status === 'pending' || effectiveJob?.status === 'running') &&
                        'text-primary',
                    )}
                  >
                    {jobStatusLabel}
                  </span>
                </div>
                <div className="flex justify-between gap-4 rounded-lg border border-border bg-background px-3 py-2">
                  <span className="text-muted-foreground">报告模板类型</span>
                  <span className="min-w-0 truncate">{reportTemplateTypeLabel}</span>
                </div>
              </div>

              <ProgressSummary
                className="mt-4"
                label="任务进度"
                percent={progressPercent}
                status={jobStatusLabel}
                tone={jobProgressTone}
              />

              {effectiveJob?.error?.message && (
                <p className="mt-4 rounded-lg bg-destructive/10 p-3 text-sm text-destructive">
                  {effectiveJob.error.message}
                </p>
              )}
              {effectiveJob?.resultSummary && (
                <p className="mt-4 rounded-lg bg-muted p-3 text-sm text-muted-foreground">
                  {effectiveJob.resultSummary}
                </p>
              )}
            </section>
          )}

          {step === 'draft' && (
            <form
              className="rounded-lg border border-border bg-card p-5"
              onSubmit={handleCreateReport}
            >
              <div className="mb-5 flex items-center gap-2">
                <FileText className="size-4 text-muted-foreground" />
                <h2 className="text-base font-semibold">创建草稿并生成大纲</h2>
              </div>

              {hasDraftPendingOutlineJob && (
                <InlineNotice className="mb-4" title="已保留报告草稿" variant="warning">
                  当前服务端报告草稿为"{currentReport?.name}"，再次提交只会复用该草稿创建大纲任务，
                  不会重复创建报告记录。
                </InlineNotice>
              )}

              <div className="grid gap-4 md:grid-cols-2">
                <label className="space-y-1.5 text-sm">
                  <span className="font-medium">报告名称</span>
                  <Input
                    maxLength={200}
                    value={form.name}
                    onChange={(event) => updateForm('name', event.target.value)}
                  />
                </label>
                <label className="space-y-1.5 text-sm">
                  <span className="font-medium">报告类型</span>
                  <Select
                    disabled={typeQuery.isLoading || typeQuery.isError || reportTypes.length === 0}
                    value={form.reportType || undefined}
                    onValueChange={(v) => {
                      const nextReportType = String(v)
                      setForm((prev) => ({
                        ...(nextReportType
                          ? applyReportTypeDraftDefaults(prev, nextReportType)
                          : { ...prev, reportType: nextReportType }),
                        templateId: '',
                      }))
                    }}
                  >
                    <SelectTrigger className="h-8 w-full">
                      <SelectValue placeholder="请选择报告类型" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="">请选择报告类型</SelectItem>
                      {reportTypes.map((type) => (
                        <SelectItem key={type.code} value={type.code}>
                          {type.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </label>
                <label className="space-y-1.5 text-sm">
                  <span className="font-medium">报告模板</span>
                  <Select
                    disabled={
                      templateQuery.isLoading || templateQuery.isError || templates.length === 0
                    }
                    value={form.templateId || undefined}
                    onValueChange={(v) => updateForm('templateId', String(v))}
                  >
                    <SelectTrigger className="h-8 w-full">
                      <SelectValue placeholder="请选择报告模板" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="">请选择报告模板</SelectItem>
                      {templates.map((template) => (
                        <SelectItem key={template.id} value={template.id}>
                          <SelectItemText>
                            {template.templateName} v{template.version}
                          </SelectItemText>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </label>
                <label className="space-y-1.5 text-sm">
                  <span className="font-medium">年份</span>
                  <Input
                    type="number"
                    min={2000}
                    max={2100}
                    value={form.year}
                    onChange={(event) => updateForm('year', Number(event.target.value))}
                  />
                </label>
                <label className="space-y-1.5 text-sm">
                  <span className="font-medium">专业</span>
                  <Input
                    maxLength={500}
                    value={form.specialty ?? ''}
                    onChange={(event) => updateForm('specialty', event.target.value)}
                  />
                </label>
                <label className="space-y-1.5 text-sm">
                  <span className="font-medium">业务对象</span>
                  <Input
                    maxLength={500}
                    value={form.businessObject ?? ''}
                    onChange={(event) => updateForm('businessObject', event.target.value)}
                  />
                </label>
                <label className="space-y-1.5 text-sm md:col-span-2">
                  <span className="font-medium">报告主题</span>
                  <Input
                    maxLength={500}
                    value={form.topic}
                    onChange={(event) => updateForm('topic', event.target.value)}
                  />
                </label>
                <label className="space-y-1.5 text-sm md:col-span-2">
                  <span className="font-medium">补充上下文 / 生成要求</span>
                  <textarea
                    className="min-h-24 w-full rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
                    maxLength={5000}
                    value={form.extraContextText ?? ''}
                    onChange={(event) => updateForm('extraContextText', event.target.value)}
                  />
                </label>
              </div>

              <div className="mt-5">
                <p className="mb-2 text-sm font-medium">引用素材</p>
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
                  <StateBlock size="compact" title="暂无可引用素材" variant="empty" />
                ) : (
                  <div className="flex flex-wrap gap-2">
                    {materials.map((material) => (
                      <button
                        key={material.id}
                        type="button"
                        className={cn(
                          'rounded-lg border px-3 py-2 text-sm transition-colors',
                          selectedMaterialIds.includes(material.id)
                            ? 'border-primary bg-primary text-primary-foreground'
                            : 'border-border bg-background text-muted-foreground hover:text-foreground',
                        )}
                        onClick={() => toggleMaterial(material.id)}
                      >
                        {material.materialName}
                      </button>
                    ))}
                  </div>
                )}
              </div>

              <div className="mt-5 flex justify-end gap-2">
                {hasDraftPendingOutlineJob && (
                  <Button
                    type="button"
                    variant="outline"
                    disabled={createReportMutation.isPending || createJobMutation.isPending}
                    onClick={handleRestartDraft}
                  >
                    <RotateCcw className="size-4" />
                    重新开始
                  </Button>
                )}
                <Button
                  type="submit"
                  disabled={
                    !canCreateReport ||
                    createReportMutation.isPending ||
                    createJobMutation.isPending
                  }
                >
                  {(createReportMutation.isPending || createJobMutation.isPending) && (
                    <Loader2 className="size-4 animate-spin" />
                  )}
                  {hasDraftPendingOutlineJob ? '复用草稿生成大纲' : '创建草稿并生成大纲'}
                </Button>
              </div>
            </form>
          )}

          {step === 'outline' && (
            <section className="rounded-lg border border-border bg-card p-5">
              <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
                <div>
                  <h2 className="text-base font-semibold">大纲章节</h2>
                  <p className="mt-1 text-sm text-muted-foreground">
                    保存整棵章节树，后端负责合法性校验和重新编号。
                  </p>
                </div>
                <div className="flex gap-2">
                  <Button
                    disabled={
                      !currentReport || outline.length === 0 || saveOutlineMutation.isPending
                    }
                    variant="outline"
                    onClick={handleSaveOutline}
                  >
                    <Save className="size-4" />
                    保存大纲
                  </Button>
                  <Button
                    disabled={
                      !currentReport ||
                      outline.length === 0 ||
                      createJobMutation.isPending ||
                      saveOutlineMutation.isPending
                    }
                    onClick={handleGenerateContent}
                  >
                    <Play className="size-4" />
                    生成正文
                  </Button>
                </div>
              </div>
              {outlinesQuery.isLoading ? (
                <StateBlock size="compact" title="大纲加载中" variant="loading" />
              ) : outlinesQuery.isError ? (
                <StateBlock
                  description={formatReportGatewayError(outlinesQuery.error, '大纲加载失败')}
                  size="compact"
                  title="大纲加载失败"
                  variant="error"
                />
              ) : outline.length === 0 ? (
                <StateBlock
                  description="大纲生成能力或数据尚未就绪时，页面不会填充本地示例。"
                  size="compact"
                  title="暂无服务端大纲"
                  variant="empty"
                />
              ) : (
                <div
                  aria-label="大纲章节列表"
                  className="space-y-2 transition-opacity duration-150"
                >
                  {flattenedOutline.map(({ node, path }) => {
                    const isDragging = pathsEqual(draggedOutlinePath, path)
                    const isDropTarget = pathsEqual(dragOverOutlinePath, path)
                    return (
                      <div
                        key={node.id ?? node.clientSectionId ?? node.title}
                        aria-label={`大纲章节：${node.title}`}
                        style={
                          node.level > 1 ? { marginLeft: `${(node.level - 1) * 2}rem` } : undefined
                        }
                        className={cn(
                          'grid grid-cols-[auto_auto_auto_minmax(0,1fr)_auto_auto] items-center gap-2 rounded-lg border border-border bg-background px-3 py-2 transition-[background-color,border-color,box-shadow,opacity,transform] duration-150 ease-out hover:bg-muted/40 focus-within:border-ring/70 focus-within:bg-muted/30',
                          isDragging && 'opacity-60',
                          isDropTarget && 'border-primary/60 bg-primary/5 shadow-sm',
                        )}
                        onDragLeave={() => {
                          if (pathsEqual(dragOverOutlinePath, path)) setDragOverOutlinePath(null)
                        }}
                        onDragOver={(event) => handleOutlineDragOver(event, path)}
                        onDrop={(event) => handleOutlineDrop(event, path)}
                      >
                        <Button
                          aria-label={`拖动章节调整顺序：${node.title}`}
                          className="cursor-grab text-muted-foreground active:cursor-grabbing"
                          draggable
                          size="icon-sm"
                          title="拖动调整同级章节顺序"
                          type="button"
                          variant="ghost"
                          onDragEnd={handleOutlineDragEnd}
                          onDragStart={(event) => handleOutlineDragStart(event, path)}
                        >
                          <GripVertical className="size-3.5" />
                        </Button>
                        <Button
                          aria-label={`在此章节后新增同级章节：${node.title}`}
                          size="icon-sm"
                          type="button"
                          variant="ghost"
                          onClick={() => handleAddOutlineSibling(path)}
                        >
                          <Plus className="size-3.5" />
                        </Button>
                        <span className="w-10 text-xs text-muted-foreground">
                          {node.numbering ?? '-'}
                        </span>
                        <Input
                          aria-label={`章节标题：${node.title}`}
                          className="h-8 min-w-0 text-sm font-medium transition-colors duration-150"
                          value={node.title}
                          onChange={(event) => handleOutlineTitleChange(path, event.target.value)}
                        />
                        <span className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground transition-colors duration-150">
                          level {node.level}
                        </span>
                        <Button
                          aria-label={`删除章节：${node.title}`}
                          size="icon-sm"
                          type="button"
                          variant="ghost"
                          onClick={() => handleDeleteOutlineNode(path)}
                        >
                          <Minus className="size-3.5" />
                        </Button>
                      </div>
                    )
                  })}
                </div>
              )}
            </section>
          )}

          {step === 'content' && (
            <section className="grid gap-4 rounded-lg border border-border bg-card p-5 lg:grid-cols-[minmax(220px,320px)_minmax(0,1fr)]">
              {sectionsQuery.isLoading ? (
                <StateBlock
                  className="lg:col-span-2"
                  size="compact"
                  title="章节加载中"
                  variant="loading"
                />
              ) : sectionsQuery.isError ? (
                <StateBlock
                  className="lg:col-span-2"
                  description={formatReportGatewayError(sectionsQuery.error, '章节加载失败')}
                  size="compact"
                  title="章节加载失败"
                  variant="error"
                />
              ) : sections.length === 0 ? (
                <StateBlock
                  className="lg:col-span-2"
                  description="正文生成能力或章节数据尚未就绪时，页面不会填充本地正文。"
                  size="compact"
                  title="暂无服务端章节"
                  variant="empty"
                />
              ) : (
                <>
                  <div aria-label="章节列表" className="min-h-0">
                    <h2 className="mb-3 text-base font-semibold">章节列表</h2>
                    <div className="max-h-[28rem] space-y-2 overflow-y-auto pr-1">
                      {sections.map((section) => (
                        <button
                          key={section.id}
                          type="button"
                          className={cn(
                            'flex w-full items-center justify-between rounded-lg border px-3 py-2 text-left text-sm',
                            activeSection?.id === section.id
                              ? 'border-primary bg-primary/10 text-primary'
                              : 'border-border bg-background text-muted-foreground hover:text-foreground',
                          )}
                          onClick={() => setActiveSectionId(section.id)}
                        >
                          <span className="min-w-0 truncate">{section.title}</span>
                          <span>{statusText[section.generationStatus]}</span>
                        </button>
                      ))}
                    </div>
                  </div>

                  <div className="flex min-h-[520px] min-w-0 flex-col">
                    <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
                      <div>
                        <h3 className="text-base font-semibold">
                          {activeSection?.title ?? '章节正文'}
                        </h3>
                        <p className="text-sm text-muted-foreground">
                          保存章节只提交结构化正文，不直接生成 DOCX。
                        </p>
                      </div>
                      <div className="flex gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => setShowVersions((prev) => !prev)}
                        >
                          版本记录{showVersions ? ' ▲' : ' ▼'}
                        </Button>
                        <Button
                          variant="outline"
                          className="text-foreground"
                          onClick={handleCancelJob}
                          disabled={!canCancelJob || cancelJobMutation.isPending}
                        >
                          {cancelJobMutation.isPending ? (
                            <Loader2 className="size-4 animate-spin" />
                          ) : (
                            <XCircle className="size-4" />
                          )}
                          取消任务
                        </Button>
                        <Button
                          variant="outline"
                          onClick={handleRetry}
                          disabled={
                            effectiveJob?.status !== 'failed' &&
                            effectiveJob?.status !== 'succeeded' &&
                            effectiveJob?.status !== 'partial_succeeded' &&
                            effectiveJob?.status !== 'canceled'
                          }
                        >
                          <RefreshCw className="size-4" />
                          重试任务
                        </Button>
                        <Button variant="outline" onClick={handleSaveSection}>
                          <PencilLine className="size-4" />
                          保存章节
                        </Button>
                        <Button onClick={handleExport}>
                          <Download className="size-4" />
                          创建 DOCX
                        </Button>
                      </div>
                    </div>

                    {showVersions && (
                      <div className="mb-4 rounded-lg border border-border bg-muted/30 p-3">
                        <h4 className="mb-2 text-sm font-medium">历史版本</h4>
                        {sectionVersionsQuery.isLoading ? (
                          <p className="text-xs text-muted-foreground">加载中...</p>
                        ) : sectionVersionsQuery.isError ? (
                          <p className="text-xs text-muted-foreground">
                            {formatReportGatewayError(
                              sectionVersionsQuery.error,
                              '章节版本加载失败',
                            )}
                          </p>
                        ) : sectionVersionsQuery.data && sectionVersionsQuery.data.length > 0 ? (
                          <div className="max-h-40 space-y-2 overflow-auto">
                            {(sectionVersionsQuery.data as ReportSectionVersion[]).map(
                              (version) => (
                                <div
                                  key={version.id}
                                  className="flex items-center justify-between rounded-lg border border-border bg-background px-3 py-2 text-xs"
                                >
                                  <div className="flex items-center gap-3">
                                    <span className="font-medium">v{version.version}</span>
                                    <span className="rounded-full bg-muted px-2 py-0.5 text-muted-foreground">
                                      {version.source === 'manual' ? '手动' : 'AI'}
                                    </span>
                                    <span className="text-muted-foreground">
                                      {formatDate(version.createdAt)}
                                    </span>
                                  </div>
                                  {version.content && (
                                    <button
                                      type="button"
                                      className="text-primary hover:underline"
                                      onClick={() => {
                                        setSectionDraft(version.content ?? '')
                                        setNotice(`已恢复版本 v${version.version} 的内容到编辑区。`)
                                      }}
                                    >
                                      恢复
                                    </button>
                                  )}
                                </div>
                              ),
                            )}
                          </div>
                        ) : (
                          <p className="text-xs text-muted-foreground">暂无历史版本。</p>
                        )}
                      </div>
                    )}

                    <textarea
                      aria-label="章节正文"
                      className="min-h-[420px] flex-1 resize-y rounded-lg border border-input bg-background px-4 py-3 text-sm leading-7 outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
                      maxLength={50000}
                      value={sectionDraft}
                      onChange={(event) => setSectionDraft(event.target.value)}
                    />
                  </div>
                </>
              )}
            </section>
          )}

          {step === 'export' && (
            <section className="rounded-lg border border-border bg-card p-5">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <h2 className="text-base font-semibold">DOCX 文件资源</h2>
                  <p className="mt-1 text-sm text-muted-foreground">
                    导出通过 POST /api/v1/report-files 创建资源；下载读取文件内容接口。
                  </p>
                </div>
                <Button onClick={handleDownload} disabled={!latestFile}>
                  <Download className="size-4" />
                  下载文件
                </Button>
              </div>

              <div className="mt-4 rounded-lg border border-border bg-background p-4">
                {latestFile ? (
                  <div className="grid gap-2 text-sm md:grid-cols-2">
                    <span className="text-muted-foreground">文件 ID</span>
                    <code>{latestFile.id}</code>
                    <span className="text-muted-foreground">文件名</span>
                    <span>{latestFile.filename ?? `${form.name}.docx`}</span>
                    <span className="text-muted-foreground">状态</span>
                    <span>{statusText[latestFile.status]}</span>
                    <span className="text-muted-foreground">创建时间</span>
                    <span>{formatDate(latestFile.createdAt)}</span>
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">
                    尚未创建导出文件。请先生成正文后创建 DOCX 文件资源。
                  </p>
                )}
              </div>
            </section>
          )}
        </div>

        <aside className="flex flex-col space-y-4">
          {user && !canManageDocumentModelSettings && (
            <section className="rounded-lg border border-border bg-card p-4">
              <h2 className="text-sm font-semibold">当前文档生成模型</h2>

              {userDocumentModelQuery.isLoading ? (
                <p className="mt-3 text-sm text-muted-foreground">加载中...</p>
              ) : userDocumentModelQuery.isError ? (
                <InlineNotice className="mt-3" title="文档生成模型加载失败" variant="error">
                  {formatReportGatewayError(userDocumentModelQuery.error, '文档生成模型加载失败')}
                </InlineNotice>
              ) : (
                <dl className="mt-3 grid gap-2 text-sm">
                  <div className="flex justify-between gap-3">
                    <dt className="text-muted-foreground">服务</dt>
                    <dd className="text-foreground">
                      {userDocumentModelQuery.data?.provider ?? 'ai-gateway'}
                    </dd>
                  </div>
                  <div className="flex justify-between gap-3">
                    <dt className="text-muted-foreground">Profile ID</dt>
                    <dd className="break-all text-foreground">
                      {userDocumentModelQuery.data?.profileId ?? '-'}
                    </dd>
                  </div>
                  <div className="flex justify-between gap-3">
                    <dt className="text-muted-foreground">模型</dt>
                    <dd className="break-all text-foreground">
                      {userDocumentModelQuery.data?.modelName ?? '-'}
                    </dd>
                  </div>
                </dl>
              )}
            </section>
          )}

          {canManageDocumentModelSettings && (
            <section className="rounded-lg border border-border bg-card p-4">
              <div className="flex items-center gap-2">
                <Settings2 className="size-4 text-muted-foreground" />
                <h2 className="text-sm font-semibold">当前文档生成模型</h2>
              </div>

              <div className="mt-3 rounded-lg border border-border bg-background p-3 text-sm">
                <div className="mb-2 font-medium text-foreground">当前生效引用</div>
                <dl className="grid gap-2">
                  <div className="flex justify-between gap-3">
                    <dt className="text-muted-foreground">服务</dt>
                    <dd className="text-foreground">ai-gateway</dd>
                  </div>
                  <div className="flex justify-between gap-3">
                    <dt className="text-muted-foreground">Profile ID</dt>
                    <dd className="break-all text-foreground">
                      {configuredDocumentProfileId || '-'}
                    </dd>
                  </div>
                  <div className="flex justify-between gap-3">
                    <dt className="text-muted-foreground">模型</dt>
                    <dd className="break-all text-foreground">{configuredDocumentModel || '-'}</dd>
                  </div>
                </dl>
              </div>

              <label className="mt-3 block space-y-1.5 text-sm">
                <span className="font-medium text-foreground">文档生成模型</span>
                <Select
                  value={documentProfileId || undefined}
                  onValueChange={(v) => handleSelectDocumentProfile(String(v))}
                  disabled={reportSettingsQuery.isLoading || chatProfilesQuery.isLoading}
                >
                  <SelectTrigger className="h-8 w-full" aria-label="文档生成模型">
                    <SelectValue placeholder="请选择聊天模型 Profile" />
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

              <div className="mt-3 space-y-2 text-sm">
                <div className="flex justify-between gap-3">
                  <span className="text-muted-foreground">待发布 Profile</span>
                  <code className="break-all">{documentProfileId || '-'}</code>
                </div>
                <div className="flex justify-between gap-3">
                  <span className="text-muted-foreground">待发布模型</span>
                  <span className="break-all">{selectedDocumentModel || '-'}</span>
                </div>
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
              {documentSettingsNotice && (
                <InlineNotice
                  className="mt-3"
                  title={documentSettingsNotice.includes('失败') ? '发布失败' : undefined}
                  variant={documentSettingsNotice.includes('失败') ? 'error' : 'info'}
                >
                  {documentSettingsNotice}
                </InlineNotice>
              )}

              <Button
                type="button"
                className="mt-3 w-full"
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
          )}
        </aside>
      </div>
    </div>
  )
}
