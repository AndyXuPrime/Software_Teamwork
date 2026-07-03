# Hook Guidelines

> Custom hooks, data fetching hooks, and streaming hooks.

## Overview

Hooks should isolate reusable stateful logic without hiding important product flow. Prefer small hooks with clear ownership over large "god hooks".

## Naming

- Use `useXxx` for all hooks.
- Use `use<Domain><Resource>Query` for TanStack Query readers.
- Use `use<Domain><Action>Mutation` for TanStack Query writers.
- Use `use<Feature>Stream` for SSE/fetch-stream logic.
- Use `use<Feature>Filters` only when filter state is reused or complex.

## Data Fetching

- Use TanStack Query for server reads and writes.
- Centralize query keys and query option factories inside the relevant feature folder.
- Mutations must invalidate or update affected queries explicitly.
- Use polling for long-running document/report tasks when SSE is not available.
- Keep generated API calls in `api/generated/`; wrap them in feature-level query functions when UI needs domain-specific behavior.

Example organization:

```txt
features/knowledge/
  knowledge.queries.ts
  knowledge.mutations.ts
  hooks/
    use-document-processing-status.ts
```

## SSE and Streaming

- Put shared stream handling in `lib/sse.ts`.
- Use `fetch` stream readers plus `AbortController`; do not use native
  `EventSource` for the main QA POST streaming path.
- QA streaming uses gateway `POST /api/v1/qa-sessions/{sessionId}/messages`
  with `Accept: text/event-stream`.
- Handle the current QA event names:
  `message.created`, `agent.iteration.started`, `reasoning.step`,
  `reasoning.delta`, `tool.started`, `tool.completed`, `tool.failed`,
  `answer.delta`, `citation.delta`, `answer.completed`, `error`, and optional
  `heartbeat`.
- Use `GET /api/v1/qa-sessions/{sessionId}/events?responseRunId=...` for
  short-term event replay and disconnect recovery.
- Streaming hooks must support cancellation through `AbortController`.
- Streaming hooks must expose enough state for UI: `status`, `content`, `progress`, `error`, and domain-specific payloads such as `citations` or generated sections.
- Never assume a stream completes successfully. Handle partial content and user cancellation.
- For QA SSE, `answer.completed` means answer generation finished, not that the
  stream has reached EOF or final persistence succeeded. Continue consuming the
  stream until EOF or a fatal `error` event; a fatal error after
  `answer.completed` must override the completed UI state and keep retry
  recovery available.
- Never expose or cache private chain-of-thought, full prompts, raw MCP tool
  parameters/results, provider raw errors, internal URLs, or storage object keys.

### Pattern: Separate transport progress from visual streaming pace

Keep the SSE parser lossless and immediate, but do not couple every
`answer.delta` to a Markdown render. The page-level stream owner must append
each delta to an in-memory complete-answer accumulator and feed a cancellable
RAF-driven display queue. The display queue may publish adaptive text batches
to UI state at roughly 20-30 FPS and accelerate when backlog grows.

```ts
fullContent += delta
streamingText.feed(delta)

// Normal terminal event: drain visible text before final UI status.
streamingText.finish()

// Stop or fatal error: cancel animation and preserve all received text.
streamingText.cancel()
patchAssistant({ content: fullContent, status: 'stopped' })
```

- `answer.completed` starts queue drain; it must not discard queued text or
  force a one-shot Markdown jump.
- User cancellation and fatal errors cancel pending RAF callbacks and commit
  the complete partial answer immediately. A delayed completion callback must
  never overwrite `stopped` or `failed`.
- Reasoning/tool/citation events remain semantically immediate. If they arrive
  in bursts, merge cumulative UI snapshots once per animation frame instead of
  applying typewriter pacing to structured events.
- Respect `prefers-reduced-motion` by consuming the current backlog in one
  frame while retaining frame-level batching.
- Keep this visual pacing outside `api/client.ts`, `api/chat.ts`, and the SSE
  parser so replay, ordering, persistence, and error semantics remain lossless.
- Unit tests must cover incremental consumption, finishing drain, cancellation,
  reduced motion, Unicode safety, and frame-batch flush/cancel behavior.

## Form Hooks

- Use React Hook Form directly in forms unless the form has reusable domain behavior.
- Keep Zod schemas next to the feature form or in `features/<domain>/schemas/`.
- Do not duplicate schema defaults between hooks and components; export default values from the schema module when needed.

## Common Mistakes

- Creating hooks that only wrap one `useState` call and are never reused.
- Hiding query invalidation inside unrelated UI components.
- Storing API responses in local component state instead of using TanStack Query.
- Forgetting cleanup for streams, polling, timers, or event listeners.
