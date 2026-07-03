const FALLBACK_APP_VERSION = '0.0.0'
const GITHUB_REPO_API_URL = 'https://api.github.com/repos/Sakayori-Iroha-168/Software_Teamwork'

export const APP_UPDATE_COMMAND = 'git fetch upstream --prune && git rebase upstream/develop'

export type AppFreshnessStatus = 'current' | 'different' | 'unknown'

export type AppFreshnessResult = {
  checkedAt: Date
  commitsAhead: number
  commitsBehind: number
  currentSha: string
  latestSha: string
  latestUrl: string | null
  status: AppFreshnessStatus
}

type GitHubCommitResponse = {
  html_url?: unknown
  sha?: unknown
}

type GitHubCompareResponse = {
  ahead_by?: unknown
  behind_by?: unknown
  status?: unknown
}

export function formatAppVersion(version: string | null | undefined) {
  const versionNumber = version?.trim().replace(/^v/i, '').trim()

  return `v${versionNumber || FALLBACK_APP_VERSION}`
}

export const appVersionLabel = formatAppVersion(__APP_VERSION__)
export const appCommitSha = __APP_COMMIT_SHA__.trim()
export const appCommitShortSha = __APP_COMMIT_SHORT_SHA__.trim() || 'unknown'

function normalizeSha(value: string) {
  return value.trim().toLowerCase()
}

function shortSha(value: string) {
  return value.slice(0, 8) || 'unknown'
}

export function compareAppFreshness(currentSha: string, latestSha: string): AppFreshnessStatus {
  const current = normalizeSha(currentSha)
  const latest = normalizeSha(latestSha)

  if (!current || !latest) return 'unknown'
  if (current === latest) return 'current'

  return 'different'
}

function parseGitHubCompareResponse(payload: unknown) {
  const comparison = payload as GitHubCompareResponse
  const commitsBehind =
    typeof comparison.ahead_by === 'number' && Number.isFinite(comparison.ahead_by)
      ? comparison.ahead_by
      : 0
  const commitsAhead =
    typeof comparison.behind_by === 'number' && Number.isFinite(comparison.behind_by)
      ? comparison.behind_by
      : 0

  return { commitsAhead, commitsBehind }
}

function parseGitHubCommitResponse(payload: unknown) {
  const commit = payload as GitHubCommitResponse
  const latestSha = typeof commit.sha === 'string' ? commit.sha : ''
  const latestUrl = typeof commit.html_url === 'string' ? commit.html_url : null

  return { latestSha, latestUrl }
}

export async function checkUpstreamDevelopFreshness(
  fetcher: typeof fetch = fetch,
  currentSha = appCommitSha,
): Promise<AppFreshnessResult> {
  const latestResponse = await fetcher(`${GITHUB_REPO_API_URL}/commits/develop`, {
    cache: 'no-store',
    headers: {
      Accept: 'application/vnd.github+json',
    },
  })

  if (!latestResponse.ok) {
    throw new Error(`GitHub 返回 ${latestResponse.status}`)
  }

  const { latestSha, latestUrl } = parseGitHubCommitResponse(await latestResponse.json())

  if (!latestSha) {
    throw new Error('GitHub 响应缺少 develop 提交 SHA')
  }

  const compareResponse = await fetcher(
    `${GITHUB_REPO_API_URL}/compare/${encodeURIComponent(currentSha)}...develop`,
    {
      cache: 'no-store',
      headers: {
        Accept: 'application/vnd.github+json',
      },
    },
  )

  if (!compareResponse.ok && compareResponse.status === 404) {
    return {
      checkedAt: new Date(),
      commitsAhead: 0,
      commitsBehind: 0,
      currentSha,
      latestSha,
      latestUrl,
      status: 'unknown',
    }
  }

  if (!compareResponse.ok) {
    throw new Error(`GitHub 返回 ${compareResponse.status}`)
  }

  const { commitsAhead, commitsBehind } = parseGitHubCompareResponse(await compareResponse.json())

  return {
    checkedAt: new Date(),
    commitsAhead,
    commitsBehind,
    currentSha,
    latestSha,
    latestUrl,
    status: commitsBehind > 0 ? 'different' : 'current',
  }
}

export function formatCommitLabel(sha: string) {
  return shortSha(sha.trim())
}
