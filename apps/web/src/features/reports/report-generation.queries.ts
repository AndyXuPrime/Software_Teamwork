import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef } from 'react'

import type { CreateReportMaterialInput, CreateReportTemplateInput } from './report-generation.api'
import {
  cancelReportJob,
  createReport,
  createReportFile,
  createReportJob,
  createReportJobAttempt,
  createReportMaterial,
  createReportTemplate,
  deleteReport,
  deleteReportMaterial,
  deleteReportTemplate,
  downloadReportFile,
  getReport,
  getReportJob,
  getReportSettings,
  getReportStatisticsOverview,
  getReportTemplateStructure,
  listDailyReportStatistics,
  listReportEvents,
  listReportMaterials,
  listReportOutlines,
  listReports,
  listReportSections,
  listReportTemplates,
  listReportTypes,
  listSectionVersions,
  updateReportOutline,
  updateReportSection,
  updateReportSettings,
  updateReportTemplateStructure,
} from './report-generation.api'
import type {
  CreateReportJobPayload,
  CreateReportPayload,
  ReportEvent,
  ReportJob,
  ReportJobStatus,
  ReportOutline,
  ReportSection,
  ReportTemplateStructure,
  UpdateReportSettingsRequest,
} from './report-generation.types'

const activeReportPollIntervalMs = 3000
const pendingReportEventPollIntervalMs = 5000
const failedReportRetryPollIntervalMs = 8000
const failedReportRetryGraceMs = 3 * 60 * 1000

const terminalReportJobStatuses = new Set<ReportJobStatus>([
  'succeeded',
  'partial_succeeded',
  'failed',
  'canceled',
])

const terminalReportEventTypes = new Set([
  'job.completed',
  'job.succeeded',
  'job.partial_succeeded',
  'job.canceled',
])

function isTerminalReportJobStatus(status: ReportJobStatus): boolean {
  return terminalReportJobStatuses.has(status)
}

function isReportContentJob(job?: Pick<ReportJob, 'jobType'> | null): boolean {
  return (
    job?.jobType === 'content_generation' ||
    job?.jobType === 'content_regeneration' ||
    job?.jobType === 'section_regeneration'
  )
}

function isReportOutlineJob(job?: Pick<ReportJob, 'jobType'> | null): boolean {
  return job?.jobType === 'outline_generation' || job?.jobType === 'outline_regeneration'
}

function resetSectionsForGeneration(sections?: ReportSection[]): ReportSection[] | undefined {
  return sections?.map((section) => ({
    ...section,
    generatedAt: undefined,
    generationStatus: 'pending',
    lastJobId: undefined,
  }))
}

function isWithinFailedReportRetryGrace(referenceTime?: string): boolean {
  if (!referenceTime) return false
  const timestamp = Date.parse(referenceTime)
  return Number.isFinite(timestamp) && Date.now() - timestamp < failedReportRetryGraceMs
}

export function getReportJobRefetchInterval(job?: ReportJob): number | false {
  if (!job) return false
  if (job.status === 'pending' || job.status === 'running') return activeReportPollIntervalMs
  if (job.status === 'failed' && isWithinFailedReportRetryGrace(job.finishedAt ?? job.createdAt)) {
    return failedReportRetryPollIntervalMs
  }
  return false
}

export function getReportSectionsRefetchInterval(
  reportId: string | null,
  job?: ReportJob | null,
): number | false {
  if (!reportId || !job || job.reportId !== reportId) return false
  if (isReportContentJob(job) && (job.status === 'pending' || job.status === 'running')) {
    return activeReportPollIntervalMs
  }
  return false
}

export function getReportEventsRefetchInterval(events?: ReportEvent[]): number | false {
  if (!events || events.length === 0) return pendingReportEventPollIntervalMs

  const latest = events[events.length - 1]
  if (!latest) return pendingReportEventPollIntervalMs
  if (terminalReportEventTypes.has(latest.eventType)) return false
  if (latest.eventType === 'job.failed') {
    return isWithinFailedReportRetryGrace(latest.createdAt)
      ? failedReportRetryPollIntervalMs
      : false
  }
  return pendingReportEventPollIntervalMs
}

export const reportKeys = {
  all: ['reports'] as const,
  types: () => [...reportKeys.all, 'types'] as const,
  templates: () => [...reportKeys.all, 'templates'] as const,
  materials: () => [...reportKeys.all, 'materials'] as const,
  records: () => [...reportKeys.all, 'records'] as const,
  recordList: (keyword: string) => [...reportKeys.records(), { keyword }] as const,
  detail: (reportId: string) => [...reportKeys.all, 'detail', reportId] as const,
  outlines: (reportId: string) => [...reportKeys.all, reportId, 'outlines'] as const,
  sections: (reportId: string) => [...reportKeys.all, reportId, 'sections'] as const,
  job: (jobId: string) => [...reportKeys.all, 'jobs', jobId] as const,
  events: (reportId: string) => [...reportKeys.all, reportId, 'events'] as const,
  sectionVersions: (reportId: string, sectionId: string) =>
    [...reportKeys.all, reportId, 'sections', sectionId, 'versions'] as const,
  stats: () => [...reportKeys.all, 'statistics'] as const,
  settings: () => [...reportKeys.all, 'settings'] as const,
  templateStructure: (templateId: string) =>
    [...reportKeys.templates(), templateId, 'structure'] as const,
}

export function useReportBootstrapQueries(reportType?: string) {
  const typeQuery = useQuery({
    queryKey: reportKeys.types(),
    queryFn: listReportTypes,
  })
  const templateQuery = useQuery({
    queryKey: [...reportKeys.templates(), { reportType }],
    queryFn: () =>
      listReportTemplates({
        reportType,
        enabled: true,
        page: 1,
        pageSize: 20,
      }),
  })
  const materialQuery = useQuery({
    queryKey: reportKeys.materials(),
    queryFn: () => listReportMaterials({ enabled: true, page: 1, pageSize: 20 }),
  })

  return { typeQuery, templateQuery, materialQuery }
}

export function useReportsQuery(keyword = '') {
  return useQuery({
    queryKey: reportKeys.recordList(keyword),
    queryFn: () => listReports({ keyword, page: 1, pageSize: 20 }),
  })
}

export function useReportDetailQueries(reportId: string | null, activeJob?: ReportJob | null) {
  const enabled = Boolean(reportId)
  const outlinesQuery = useQuery({
    queryKey: reportKeys.outlines(reportId ?? ''),
    queryFn: () => listReportOutlines(reportId ?? ''),
    enabled,
  })
  const sectionsQuery = useQuery({
    queryKey: reportKeys.sections(reportId ?? ''),
    queryFn: () => listReportSections(reportId ?? ''),
    enabled,
    refetchInterval: () => getReportSectionsRefetchInterval(reportId, activeJob),
  })

  return { outlinesQuery, sectionsQuery }
}

export function useReportJobQuery(jobId: string | null) {
  const queryClient = useQueryClient()
  const refreshedTerminalJobsRef = useRef(new Set<string>())
  const refreshedActiveJobsRef = useRef(new Set<string>())
  const query = useQuery({
    queryKey: reportKeys.job(jobId ?? ''),
    queryFn: () => getReportJob(jobId ?? ''),
    enabled: Boolean(jobId),
    refetchInterval: (query) => {
      return getReportJobRefetchInterval(query.state.data)
    },
    structuralSharing: false,
  })

  useEffect(() => {
    const job = query.data
    if (!job?.reportId) return

    if (job.status === 'pending' || job.status === 'running') {
      const refreshKey = `${job.id}:${job.status}:${query.dataUpdatedAt}`
      if (refreshedActiveJobsRef.current.has(refreshKey)) return
      refreshedActiveJobsRef.current.add(refreshKey)

      if (isReportContentJob(job)) {
        void queryClient.invalidateQueries({ queryKey: reportKeys.sections(job.reportId) })
      }
      void queryClient.invalidateQueries({ queryKey: reportKeys.events(job.reportId) })
      return
    }

    if (!isTerminalReportJobStatus(job.status)) return

    const refreshKey = `${job.id}:${job.status}:${job.finishedAt ?? ''}`
    if (refreshedTerminalJobsRef.current.has(refreshKey)) return
    refreshedTerminalJobsRef.current.add(refreshKey)

    void queryClient.invalidateQueries({ queryKey: reportKeys.outlines(job.reportId) })
    void queryClient.invalidateQueries({ queryKey: reportKeys.sections(job.reportId) })
    void queryClient.invalidateQueries({ queryKey: reportKeys.detail(job.reportId) })
    void queryClient.invalidateQueries({ queryKey: reportKeys.records() })
    void queryClient.invalidateQueries({ queryKey: reportKeys.events(job.reportId) })
  }, [query.data, query.dataUpdatedAt, queryClient])

  return query
}

export function useCreateReportMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (payload: CreateReportPayload) => createReport(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: reportKeys.records() })
    },
  })
}

export function useCreateReportJobMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ reportId, payload }: { reportId: string; payload: CreateReportJobPayload }) =>
      createReportJob(reportId, payload),
    onMutate: ({ payload, reportId }) => {
      if (isReportOutlineJob(payload)) {
        queryClient.setQueryData<ReportOutline[]>(reportKeys.outlines(reportId), [])
        queryClient.setQueryData<ReportSection[]>(reportKeys.sections(reportId), [])
        return
      }

      if (isReportContentJob(payload)) {
        queryClient.setQueryData<ReportSection[]>(reportKeys.sections(reportId), (sections) =>
          resetSectionsForGeneration(sections),
        )
      }
    },
    onSuccess: (job) => {
      if (isReportOutlineJob(job)) {
        if (isTerminalReportJobStatus(job.status)) {
          void queryClient.invalidateQueries({
            queryKey: reportKeys.outlines(job.reportId),
          })
          void queryClient.invalidateQueries({
            queryKey: reportKeys.sections(job.reportId),
          })
        }
      } else {
        void queryClient.invalidateQueries({
          queryKey: reportKeys.outlines(job.reportId),
        })
        void queryClient.invalidateQueries({
          queryKey: reportKeys.sections(job.reportId),
        })
      }
      void queryClient.invalidateQueries({ queryKey: reportKeys.records() })
    },
  })
}

export function useUpdateReportOutlineMutation(reportId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      outlineId,
      sections,
    }: {
      outlineId: string
      sections: ReportOutline['sections']
    }) => updateReportOutline(reportId, outlineId, sections),
    onSuccess: (outline) => {
      queryClient.setQueryData<ReportOutline[]>(reportKeys.outlines(reportId), (outlines) => {
        const existing = outlines ?? []
        const next = existing.filter((item) => item.id !== outline.id)
        return [outline, ...next]
      })
      void queryClient.invalidateQueries({
        queryKey: reportKeys.outlines(reportId),
      })
      void queryClient.invalidateQueries({
        queryKey: reportKeys.sections(reportId),
      })
    },
  })
}

export function useUpdateReportSectionMutation(reportId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      sectionId,
      title,
      content,
    }: {
      sectionId: string
      title?: string
      content?: string
    }) => updateReportSection(reportId, sectionId, { title, content }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: reportKeys.sections(reportId),
      })
    },
  })
}

export function useCreateReportFileMutation() {
  return useMutation({
    mutationFn: createReportFile,
  })
}

export function useRetryReportJobMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ jobId }: { jobId: string; reportId?: string }) => createReportJobAttempt(jobId),
    onSuccess: (attempt, variables) => {
      queryClient.setQueryData<ReportJob>(reportKeys.job(attempt.jobId), (job) =>
        job
          ? {
              ...job,
              error: undefined,
              finishedAt: undefined,
              progress: {},
              resultSummary: undefined,
              startedAt: undefined,
              status: 'pending',
            }
          : job,
      )
      void queryClient.invalidateQueries({ queryKey: reportKeys.job(attempt.jobId) })
      if (variables.reportId) {
        queryClient.setQueryData<ReportSection[]>(
          reportKeys.sections(variables.reportId),
          (sections) => resetSectionsForGeneration(sections),
        )
        void queryClient.invalidateQueries({
          queryKey: reportKeys.events(variables.reportId),
        })
      }
    },
  })
}

export function useDownloadReportFileMutation() {
  return useMutation({
    mutationFn: (reportFileId: string) => downloadReportFile(reportFileId),
  })
}

export function useReportStatisticsQueries() {
  const overviewQuery = useQuery({
    queryKey: reportKeys.stats(),
    queryFn: getReportStatisticsOverview,
  })
  const dailyQuery = useQuery({
    queryKey: [...reportKeys.stats(), 'daily'],
    queryFn: () => listDailyReportStatistics(30),
  })

  return { overviewQuery, dailyQuery }
}

type UseReportSettingsQueryOptions = {
  enabled?: boolean
}

export function useReportSettingsQuery(options: UseReportSettingsQueryOptions = {}) {
  return useQuery({
    queryKey: reportKeys.settings(),
    queryFn: getReportSettings,
    enabled: options.enabled ?? true,
  })
}

export function useUpdateReportSettingsMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (payload: UpdateReportSettingsRequest) => updateReportSettings(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: reportKeys.settings() })
    },
  })
}

export function useReport(reportId: string | null) {
  return useQuery({
    queryKey: reportKeys.detail(reportId ?? ''),
    queryFn: () => getReport(reportId ?? ''),
    enabled: Boolean(reportId),
  })
}

export function useDeleteReport() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (reportId: string) => deleteReport(reportId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: reportKeys.records() })
    },
  })
}

export function useTemplateStructure(templateId: string | null) {
  return useQuery({
    queryKey: reportKeys.templateStructure(templateId ?? ''),
    queryFn: () => getReportTemplateStructure(templateId ?? ''),
    enabled: Boolean(templateId),
  })
}

export function useUpdateTemplateStructure(templateId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (payload: ReportTemplateStructure) =>
      updateReportTemplateStructure(templateId, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: reportKeys.templateStructure(templateId),
      })
    },
  })
}

export function useCreateTemplate() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (payload: CreateReportTemplateInput) => createReportTemplate(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: reportKeys.templates() })
      void queryClient.invalidateQueries({ queryKey: reportKeys.stats() })
    },
  })
}

export function useCreateMaterial() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (payload: CreateReportMaterialInput) => createReportMaterial(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: reportKeys.materials() })
      void queryClient.invalidateQueries({ queryKey: reportKeys.stats() })
    },
  })
}

export function useDeleteTemplate() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (templateId: string) => deleteReportTemplate(templateId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: reportKeys.templates() })
    },
  })
}

export function useDeleteMaterial() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (materialId: string) => deleteReportMaterial(materialId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: reportKeys.materials() })
      void queryClient.invalidateQueries({ queryKey: reportKeys.stats() })
    },
  })
}

export function useReportEvents(reportId: string | null) {
  return useQuery({
    queryKey: reportKeys.events(reportId ?? ''),
    queryFn: () => listReportEvents(reportId ?? ''),
    enabled: Boolean(reportId),
    refetchInterval: (query) => {
      return getReportEventsRefetchInterval(query.state.data)
    },
    select: (data) => data,
  })
}

export function useCancelReportJob() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (jobId: string) => cancelReportJob(jobId),
    onSuccess: (job) => {
      queryClient.setQueryData<ReportJob>(reportKeys.job(job.id), job)
      void queryClient.invalidateQueries({ queryKey: reportKeys.job(job.id) })
      if (job.reportId) {
        void queryClient.invalidateQueries({
          queryKey: reportKeys.events(job.reportId),
        })
      }
    },
  })
}

export function useSectionVersions(reportId: string | null, sectionId: string | null) {
  return useQuery({
    queryKey: reportKeys.sectionVersions(reportId ?? '', sectionId ?? ''),
    queryFn: () => listSectionVersions(reportId ?? '', sectionId ?? ''),
    enabled: Boolean(reportId) && Boolean(sectionId),
  })
}
