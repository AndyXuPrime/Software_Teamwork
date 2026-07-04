import {
  buildQuery,
  gatewayFileRequest,
  gatewayPageRequest,
  gatewayRequest,
  requestVoid,
} from '@/api/client'

import type {
  CreateReportJobPayload,
  CreateReportPayload,
  Report,
  ReportDailyStatistic,
  ReportEvent,
  ReportFile,
  ReportJob,
  ReportJobAttempt,
  ReportMaterial,
  ReportOutline,
  ReportSection,
  ReportSectionVersion,
  ReportSettings,
  ReportSettingsUpdateResult,
  ReportStatisticsOverview,
  ReportStatus,
  ReportTemplate,
  ReportTemplateStructure,
  ReportType,
  ReportTypeCode,
  UpdateReportSettingsRequest,
} from './report-generation.types'

export type ReportListParams = {
  page?: number
  pageSize?: number
  reportType?: ReportTypeCode
  status?: ReportStatus | string
  keyword?: string
}

export type ReportTemplateListParams = {
  page?: number
  pageSize?: number
  reportType?: ReportTypeCode
  enabled?: boolean
}

export type CreateReportTemplateInput = {
  description?: string
  file: File
  reportType: string
  templateName: string
}

export type ReportMaterialListParams = {
  page?: number
  pageSize?: number
  category?: string
  enabled?: boolean
}

export type CreateReportMaterialInput = {
  category?: string
  description?: string
  file: File
  materialName: string
  materialType: string
  tags?: string[]
}

export function listReportTypes(): Promise<ReportType[]> {
  return gatewayRequest<ReportType[]>('/report-types')
}

export function listReportTemplates(params: ReportTemplateListParams = {}) {
  return gatewayPageRequest<ReportTemplate>(`/report-templates${buildQuery(params)}`)
}

export function createReportTemplate(input: CreateReportTemplateInput): Promise<ReportTemplate> {
  const formData = new FormData()
  formData.append('file', input.file, input.file.name)
  formData.append('templateName', input.templateName)
  formData.append('reportType', input.reportType)
  if (input.description) formData.append('description', input.description)

  return gatewayRequest<ReportTemplate>('/report-templates', {
    method: 'POST',
    body: formData,
  })
}

export function listReportMaterials(params: ReportMaterialListParams = {}) {
  return gatewayPageRequest<ReportMaterial>(`/report-materials${buildQuery(params)}`)
}

export function createReportMaterial(input: CreateReportMaterialInput): Promise<ReportMaterial> {
  const formData = new FormData()
  formData.append('file', input.file, input.file.name)
  formData.append('materialName', input.materialName)
  formData.append('materialType', input.materialType)
  if (input.category) formData.append('category', input.category)
  if (input.description) formData.append('description', input.description)
  if (input.tags && input.tags.length > 0) formData.append('tags', input.tags.join(','))

  return gatewayRequest<ReportMaterial>('/report-materials', {
    method: 'POST',
    body: formData,
  })
}

export function createReport(payload: CreateReportPayload): Promise<Report> {
  return gatewayRequest<Report>('/reports', {
    method: 'POST',
    body: payload,
  })
}

export function listReports(params: ReportListParams = {}) {
  return gatewayPageRequest<Report>(`/reports${buildQuery(params)}`)
}

export function listReportOutlines(reportId: string): Promise<ReportOutline[]> {
  return gatewayRequest<ReportOutline[]>(`/reports/${encodeURIComponent(reportId)}/outlines`)
}

export function updateReportOutline(
  reportId: string,
  outlineId: string,
  sections: ReportOutline['sections'],
): Promise<ReportOutline> {
  return gatewayRequest<ReportOutline>(
    `/reports/${encodeURIComponent(reportId)}/outlines/${encodeURIComponent(outlineId)}`,
    {
      method: 'PATCH',
      body: { sections, manualEdited: true },
    },
  )
}

export function listReportSections(reportId: string): Promise<ReportSection[]> {
  return gatewayRequest<ReportSection[]>(`/reports/${encodeURIComponent(reportId)}/sections`)
}

export function updateReportSection(
  reportId: string,
  sectionId: string,
  payload: { title?: string; content?: string; tables?: Record<string, unknown>[] },
): Promise<ReportSection> {
  return gatewayRequest<ReportSection>(
    `/reports/${encodeURIComponent(reportId)}/sections/${encodeURIComponent(sectionId)}`,
    {
      method: 'PATCH',
      body: { ...payload, manualEdited: true },
    },
  )
}

export function createReportJob(
  reportId: string,
  payload: CreateReportJobPayload,
): Promise<ReportJob> {
  return gatewayRequest<ReportJob>(`/reports/${encodeURIComponent(reportId)}/jobs`, {
    method: 'POST',
    body: payload,
  })
}

export function getReportJob(jobId: string): Promise<ReportJob> {
  return gatewayRequest<ReportJob>(`/report-jobs/${encodeURIComponent(jobId)}`)
}

export function createReportJobAttempt(jobId: string): Promise<ReportJobAttempt> {
  return gatewayRequest<ReportJobAttempt>(`/report-jobs/${encodeURIComponent(jobId)}/attempts`, {
    method: 'POST',
    body: { reason: 'frontend_retry' },
  })
}

export function createReportFile(payload: {
  reportId: string
  format: 'docx'
  templateId?: string
  styleOptions?: Record<string, unknown>
}): Promise<ReportFile> {
  return gatewayRequest<ReportFile>('/report-files', {
    method: 'POST',
    body: payload,
  })
}

export function downloadReportFile(reportFileId: string): Promise<Blob> {
  return gatewayFileRequest(`/report-files/${encodeURIComponent(reportFileId)}/content`)
}

export function getReportStatisticsOverview(): Promise<ReportStatisticsOverview> {
  return gatewayRequest<ReportStatisticsOverview>('/report-statistics/overview')
}

export function getReportSettings(): Promise<ReportSettings> {
  return gatewayRequest<ReportSettings>('/report-settings')
}

export function updateReportSettings(
  payload: UpdateReportSettingsRequest,
): Promise<ReportSettingsUpdateResult> {
  return gatewayRequest<ReportSettingsUpdateResult>('/report-settings', {
    method: 'PATCH',
    body: payload,
  })
}

export function listDailyReportStatistics(days = 30): Promise<ReportDailyStatistic[]> {
  return gatewayRequest<ReportDailyStatistic[]>(`/report-statistics/daily${buildQuery({ days })}`)
}

export function getReport(reportId: string): Promise<Report> {
  return gatewayRequest<Report>(`/reports/${encodeURIComponent(reportId)}`)
}

export function deleteReport(reportId: string): Promise<void> {
  return requestVoid(`/reports/${encodeURIComponent(reportId)}`, {
    method: 'DELETE',
  })
}

export function getReportTemplateStructure(templateId: string): Promise<ReportTemplateStructure> {
  return gatewayRequest<ReportTemplateStructure>(
    `/report-templates/${encodeURIComponent(templateId)}/structure`,
  )
}

export function updateReportTemplateStructure(
  templateId: string,
  payload: ReportTemplateStructure,
): Promise<ReportTemplateStructure> {
  return gatewayRequest<ReportTemplateStructure>(
    `/report-templates/${encodeURIComponent(templateId)}/structure`,
    {
      method: 'PATCH',
      body: payload,
    },
  )
}

export function deleteReportTemplate(templateId: string): Promise<void> {
  return requestVoid(`/report-templates/${encodeURIComponent(templateId)}`, {
    method: 'DELETE',
  })
}

export function deleteReportMaterial(materialId: string): Promise<void> {
  return requestVoid(`/report-materials/${encodeURIComponent(materialId)}`, {
    method: 'DELETE',
  })
}

export function listReportEvents(reportId: string): Promise<ReportEvent[]> {
  return gatewayRequest<ReportEvent[]>(`/reports/${encodeURIComponent(reportId)}/events`)
}

export function cancelReportJob(jobId: string): Promise<ReportJob> {
  return gatewayRequest<ReportJob>(`/report-jobs/${encodeURIComponent(jobId)}`, {
    method: 'PATCH',
    body: { status: 'canceled' },
  })
}

export function listSectionVersions(
  reportId: string,
  sectionId: string,
): Promise<ReportSectionVersion[]> {
  return gatewayRequest<ReportSectionVersion[]>(
    `/reports/${encodeURIComponent(reportId)}/sections/${encodeURIComponent(sectionId)}/versions`,
  )
}
