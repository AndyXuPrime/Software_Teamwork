import { describe, expect, it } from 'vitest'

import { checkUpstreamDevelopFreshness } from './app-version'

describe('app version freshness', () => {
  it('checks develop freshness without browser caching and returns behind commits', async () => {
    const requests: Array<{ input: RequestInfo | URL; init?: RequestInit }> = []
    const fetcher: typeof fetch = async (input, init) => {
      requests.push({ input, init })

      const url = String(input)
      if (url.endsWith('/commits/develop')) {
        return new Response(
          JSON.stringify({
            html_url: 'https://github.com/Sakayori-Iroha-168/Software_Teamwork/commit/develop',
            sha: '2222222222222222222222222222222222222222',
          }),
          { status: 200 },
        )
      }

      return new Response(
        JSON.stringify({
          ahead_by: 3,
          behind_by: 1,
        }),
        { status: 200 },
      )
    }

    const result = await checkUpstreamDevelopFreshness(
      fetcher,
      '1111111111111111111111111111111111111111',
    )

    expect(result.status).toBe('different')
    expect(result.commitsBehind).toBe(3)
    expect(result.commitsAhead).toBe(1)
    expect(result.latestSha).toBe('2222222222222222222222222222222222222222')
    expect(requests).toHaveLength(2)
    expect(requests.every((request) => request.init?.cache === 'no-store')).toBe(true)
  })

  it('returns unknown when the local commit is not available to GitHub compare', async () => {
    const fetcher: typeof fetch = async (input) => {
      const url = String(input)
      if (url.endsWith('/commits/develop')) {
        return new Response(
          JSON.stringify({
            html_url: 'https://github.com/Sakayori-Iroha-168/Software_Teamwork/commit/develop',
            sha: '2222222222222222222222222222222222222222',
          }),
          { status: 200 },
        )
      }

      return new Response(JSON.stringify({ message: 'Not Found' }), { status: 404 })
    }

    const result = await checkUpstreamDevelopFreshness(
      fetcher,
      'local-only-local-only-local-only-local-only',
    )

    expect(result.status).toBe('unknown')
    expect(result.commitsBehind).toBe(0)
  })
})
