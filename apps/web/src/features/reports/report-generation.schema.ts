import { z } from 'zod'

export const createReportSchema = z.object({
  name: z.string().trim().min(1, '请输入报告名称'),
  reportType: z.string().trim().min(1, '请选择报告类型'),
  templateId: z.string().trim().min(1, '请选择报告模板'),
  topic: z.string().trim().min(1, '请输入报告主题'),
  specialty: z.string().trim().optional(),
  businessObject: z.string().trim().optional(),
  year: z.coerce
    .number()
    .int('年份必须是整数')
    .min(2000, '年份不能早于 2000')
    .max(2100, '年份不能晚于 2100'),
  extraContextText: z.string().trim().optional(),
})

export type CreateReportFormValues = z.infer<typeof createReportSchema>

const defaultReportTypeCode = 'summer_peak_inspection'

type ReportTypeDraftDefaults = Pick<
  CreateReportFormValues,
  'businessObject' | 'extraContextText' | 'name' | 'specialty' | 'topic'
>

const defaultReportTypeDraftDefaults: ReportTypeDraftDefaults = {
  name: '2026年迎峰度夏检查报告',
  topic: '迎峰度夏设备安全检查',
  specialty: '电气一次',
  businessObject: '主变、厂用电系统、保护装置',
  extraContextText: '重点关注高温高负荷、缺陷闭环、应急保障和历史隐患治理。',
}

const reportTypeDraftDefaults: Record<string, ReportTypeDraftDefaults> = {
  summer_peak_inspection: defaultReportTypeDraftDefaults,
  coal_inventory_audit: {
    name: '2026年煤库存审计报告',
    topic: '煤场库存账实与保供风险审计',
    specialty: '燃料管理',
    businessObject: '煤场、入厂煤计量、采制化系统',
    extraContextText: '重点关注账实差异、煤质计量、库存周转、保供风险和整改闭环。',
  },
}

export const defaultCreateReportValues: CreateReportFormValues = {
  ...defaultReportTypeDraftDefaults,
  reportType: defaultReportTypeCode,
  templateId: '',
  year: new Date().getFullYear(),
}

export function getReportTypeDraftDefaults(reportType: string): ReportTypeDraftDefaults {
  return reportTypeDraftDefaults[reportType] ?? defaultReportTypeDraftDefaults
}

export function getCreateReportDefaults(
  reportType = defaultReportTypeCode,
): CreateReportFormValues {
  return {
    ...defaultCreateReportValues,
    ...getReportTypeDraftDefaults(reportType),
    reportType,
    templateId: '',
    year: new Date().getFullYear(),
  }
}

export function isReportTypeDraftDefaultValue(
  field: keyof ReportTypeDraftDefaults,
  value: string | undefined,
): boolean {
  return Object.values(reportTypeDraftDefaults).some((defaults) => defaults[field] === value)
}
