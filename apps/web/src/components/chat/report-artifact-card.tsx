import { Download, ExternalLink, FileText } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import type { QAReportArtifact } from '@/lib/types'
import { cn } from '@/lib/utils'

type ReportArtifactCardProps = {
  artifact: QAReportArtifact
  onDownload?: (reportFileId: string, filename: string) => void
}

function getJobStatusVariant(
  status: string | undefined,
): 'default' | 'secondary' | 'destructive' | 'outline' {
  if (status === 'succeeded' || status === 'completed') return 'default'
  if (status === 'failed') return 'destructive'
  return 'secondary'
}

function getJobStatusLabel(status: string | undefined): string {
  switch (status) {
    case 'accepted':
      return '已接受'
    case 'pending':
      return '等待中'
    case 'running':
      return '生成中'
    case 'succeeded':
      return '已完成'
    case 'completed':
      return '已完成'
    case 'failed':
      return '失败'
    case 'canceled':
      return '已取消'
    default:
      return status ?? '处理中'
  }
}

function isJobRunning(status: string | undefined): boolean {
  return status === 'running' || status === 'accepted' || status === 'pending'
}

function isJobFailed(status: string | undefined): boolean {
  return status === 'failed'
}

function canDownload(artifact: QAReportArtifact): boolean {
  return (
    artifact.fileStatus === 'succeeded' &&
    Boolean(artifact.reportFileId) &&
    Boolean(artifact.downloadPath)
  )
}

const MAX_TITLES = 5
const MAX_SUMMARY_LENGTH = 120

export default function ReportArtifactCard({ artifact, onDownload }: ReportArtifactCardProps) {
  const jobStatus = artifact.jobStatus
  const reportName = artifact.reportName ?? '报告生成'
  const preview = artifact.preview

  // Collect display titles from preview
  const titles = preview?.outlineTitles ?? preview?.sectionTitles ?? []
  const displayTitles = titles.slice(0, MAX_TITLES)
  const remaining = titles.length - MAX_TITLES

  const summary = preview?.summary ?? ''
  const truncatedSummary =
    summary.length > MAX_SUMMARY_LENGTH ? summary.slice(0, MAX_SUMMARY_LENGTH) + '…' : summary

  const handleDownload = () => {
    if (!canDownload(artifact) || !artifact.reportFileId) return
    onDownload?.(artifact.reportFileId, artifact.filename ?? 'report.docx')
  }

  const reportUrl = artifact.reportId
    ? `/reports/records?reportId=${encodeURIComponent(artifact.reportId)}`
    : null

  return (
    <div className="mt-3 overflow-hidden rounded-lg border border-border bg-card shadow-sm transition-shadow hover:shadow-md">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-border/50 px-4 py-3">
        <div className="flex items-center gap-2 min-w-0">
          <FileText className="size-4 shrink-0 text-muted-foreground" />
          <span className="truncate text-sm font-medium">{reportName}</span>
          {preview?.statusText && (
            <span className="truncate text-xs text-muted-foreground">{preview.statusText}</span>
          )}
        </div>
        {jobStatus && (
          <Badge
            variant={getJobStatusVariant(jobStatus)}
            className={cn('ml-2 shrink-0', isJobRunning(jobStatus) && 'animate-pulse')}
          >
            {getJobStatusLabel(jobStatus)}
          </Badge>
        )}
      </div>

      {/* Preview body */}
      {(displayTitles.length > 0 || truncatedSummary || preview?.progressPercent != null) && (
        <div className="px-4 py-3 space-y-2">
          {/* Progress bar */}
          {preview?.progressPercent != null && (
            <div className="w-full">
              <div className="mb-1 flex items-center justify-between text-xs text-muted-foreground">
                <span>进度</span>
                <span>{preview.progressPercent}%</span>
              </div>
              <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
                <div
                  className={cn(
                    'h-full rounded-full transition-all duration-500',
                    isJobFailed(jobStatus) ? 'bg-destructive' : 'bg-primary',
                  )}
                  style={{ width: `${Math.min(100, Math.max(0, preview.progressPercent))}%` }}
                />
              </div>
            </div>
          )}

          {/* Titles */}
          {displayTitles.length > 0 && (
            <div className="space-y-0.5">
              {displayTitles.map((t, i) => (
                <div key={i} className="flex items-start gap-1.5 text-xs text-muted-foreground">
                  <span className="mt-0.5 size-1 shrink-0 rounded-full bg-muted-foreground/40" />
                  <span className="truncate">{t}</span>
                </div>
              ))}
              {remaining > 0 && (
                <div className="text-xs text-muted-foreground pl-3.5">...等 {remaining} 项</div>
              )}
            </div>
          )}

          {/* Summary */}
          {truncatedSummary && (
            <p className="text-xs leading-relaxed text-muted-foreground">{truncatedSummary}</p>
          )}
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center justify-end gap-2 border-t border-border/50 px-4 py-2">
        {reportUrl && (
          <a
            href={reportUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground"
          >
            <ExternalLink className="size-3" />
            查看详情
          </a>
        )}
        <Button
          variant="outline"
          size="sm"
          disabled={!canDownload(artifact)}
          onClick={handleDownload}
          className="h-7 px-2 text-xs transition-all hover:bg-primary hover:text-primary-foreground hover:scale-105 active:scale-95"
        >
          <Download className="mr-1 size-3" />
          下载报告
        </Button>
      </div>
    </div>
  )
}
