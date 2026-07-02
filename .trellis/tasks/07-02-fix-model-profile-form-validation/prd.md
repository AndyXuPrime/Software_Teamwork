# Fix model profile form validation

## Goal

Resolve GitHub issue #445 by aligning the model profile create/edit form with
the public Gateway and AI Gateway validation contract so admins can save chat,
embedding, and rerank model profiles without the frontend submitting a default
`defaultParameters.max_tokens` value that AI Gateway rejects.

## Confirmed Facts

- Issue #445 is open, assigned to `Jackeyliu37`, labeled `bug`, `frontend`,
  `service:gateway`, and `service:ai-gateway`, and recommends branch
  `Frontend/fix/model-profile-form-validation`.
- The issue description matches local reproduction: `POST
  /api/v1/admin/model-profiles` fails with `validation_error` and field
  `defaultParameters: must not contain sensitive keys` when the request body
  includes `defaultParameters.max_tokens`.
- The issue body still references `upstream/develop @ f70652e`; after
  `git fetch upstream --prune` on 2026-07-02, current `upstream/develop` is
  `58d39d4`. The defect remains reproducible on that newer base.
- Frontend code in `apps/web/src/pages/admin/model-profiles.tsx` currently
  emits `defaultParameters: { max_tokens: form.maxTokens }` for create and
  update payloads.
- Gateway OpenAPI documents `defaultParameters` as purpose-specific provider
  parameters and examples include `max_tokens`, but issue #445 explicitly
  scopes this task to frontend payload/error handling and says not to relax AI
  Gateway sensitive-key validation in this task.
- AI Gateway requires `dimensions > 0` for embedding profiles and `topN > 0`
  for rerank profiles.
- The local seeded admin has `admin:model-profile:write`; the observed failure
  is not an authorization problem.

## Requirements

- Create/update payloads must omit `defaultParameters.max_tokens` when the max
  token input is empty or zero.
- The form must not send known AI Gateway sensitive keys through
  `defaultParameters` as part of this fix.
- Create validation must require name, purpose, provider, base URL, model, and
  API key, plus `dimensions` for embedding and `topN` for rerank.
- Edit validation must require name, provider, base URL, model, plus
  `dimensions` for embedding and `topN` for rerank.
- Gateway `ApiError` field details and `requestId` must be visible in the model
  profile form failure notification.
- The implementation must keep browser traffic on Gateway `/api/v1/**` only.
- The implementation must remain scoped to the model profile page/hooks/tests
  unless a directly necessary helper extraction is justified.

## Acceptance Criteria

- [ ] Frontend tests cover that create/update payloads omit default
  `max_tokens` when max token value is `0`.
- [ ] Frontend tests cover that positive max token values are either not emitted
  by default or are handled according to the final scoped behavior without
  triggering AI Gateway's current sensitive-key rejection.
- [ ] Frontend tests cover create validation for missing embedding dimensions
  and missing rerank TopN.
- [ ] Frontend tests or component-level tests cover that API errors show the
  normalized message plus `requestId` when present.
- [ ] Manual or API verification confirms chat profile creation through
  `/api/v1/admin/model-profiles` no longer fails because of default
  `max_tokens`.
- [ ] `bun run --cwd apps/web check` passes.
- [ ] `bun run --cwd apps/web build` passes.
- [ ] `git diff --check` passes.

## Out Of Scope

- Do not loosen AI Gateway sensitive-key validation in this task.
- Do not implement generic arbitrary provider-parameter editing.
- Do not handle issue #442 model profile enable/disable interactions unless a
  direct conflict appears.
- Do not run or record real provider smoke tests, and do not expose real API
  keys, service tokens, prompts, document text, embeddings, object keys, or
  provider raw responses.

