import type { ReportMaterial } from './report-generation.types'

const legacyLocalSeedMaterialId = '22222222-2222-4222-8222-222222222201'

const realisticSeedMaterialDisplay = {
  category: '煤库存审计',
  description:
    '记录2024年12月31日煤场库存盘点口径、地磅计量差异、热值折算和保供库存预警阈值，供煤库存审计报告正文引用。',
  filename: '煤场库存盘点工作底稿.md',
  materialName: '煤场库存盘点工作底稿',
  materialType: '审计底稿',
  tags: ['煤场库存', '盘点差异', '热值折算', '保供风险'],
}

export type ReportMaterialDisplayDetails = {
  category: string
  createdAt: string
  description: string
  enabledText: string
  filename: string
  materialName: string
  materialType: string
  tags: string
}

function formatMaterialValue(value: string | undefined | null) {
  return value?.trim() || '未填写'
}

function formatMaterialType(value: string | undefined | null) {
  const normalized = value?.trim()
  if (!normalized) return '未填写'
  const labels: Record<string, string> = {
    audit_workpaper: '审计底稿',
    plant_report: '运行报告',
    technical_doc: '技术资料',
    text: '文本资料',
  }
  return labels[normalized] ?? normalized
}

function formatMaterialTags(tags: string[] | undefined | null) {
  return tags && tags.length > 0 ? tags.join('、') : '未填写'
}

function formatMaterialCreatedAt(value: string | undefined | null) {
  if (!value) return '未填写'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)
}

export function isLegacyLocalSeedReportMaterial(material: ReportMaterial) {
  return (
    material.id === legacyLocalSeedMaterialId ||
    material.filename === 'local-demo-inspection-notes.md' ||
    material.materialName === '本地演示检查记录'
  )
}

export function getReportMaterialDisplayDetails(
  material: ReportMaterial,
): ReportMaterialDisplayDetails {
  if (isLegacyLocalSeedReportMaterial(material)) {
    return {
      ...realisticSeedMaterialDisplay,
      createdAt: formatMaterialCreatedAt(material.createdAt),
      enabledText: material.enabled ? '可引用' : '停用',
      tags: formatMaterialTags(realisticSeedMaterialDisplay.tags),
    }
  }

  return {
    category: formatMaterialValue(material.category),
    createdAt: formatMaterialCreatedAt(material.createdAt),
    description: formatMaterialValue(material.description),
    enabledText: material.enabled ? '可引用' : '停用',
    filename: formatMaterialValue(material.filename),
    materialName: formatMaterialValue(material.materialName),
    materialType: formatMaterialType(material.materialType),
    tags: formatMaterialTags(material.tags),
  }
}
