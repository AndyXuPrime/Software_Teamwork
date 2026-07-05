# Design

## CI Detection

The workflow already triggers for `deploy/**`; the missing piece is the
`has-policy-check` output. Add an explicit approved-cloud-support-file pattern
for `deploy/docker/full/**` and include it in `dockerPolicyChanged`.

This should remain separate from `dockerArtifactPattern`, which intentionally
flags non-approved business Docker artifacts. `deploy/docker/full/**` is an
approved exception, but changes there still need policy and compose checks.

## Seed-Disabled Model Fallback

Cloud runtime overrides should be explicit:

- QA gets `MODEL_ID` only from `${MODEL_ID:-}`.
- Document gets `DOCUMENT_AI_GATEWAY_MODEL` only from
  `${DOCUMENT_AI_GATEWAY_MODEL:-}`.
- `AI_GATEWAY_LOCAL_CHAT_MODEL` remains a seed-only input used by the seed
  helper when `DOCKER_SEED_ENABLED=true` and provider seed is enabled.

This preserves the default behavior of using AI Gateway profile configuration
without accidentally passing `<cloud-chat-model>` to Document/QA.

## Tests

Add a small script/unit test for Docker workflow detection if no helper exists.
It should evaluate the embedded GitHub script behavior for representative file
sets, including `deploy/docker/full/go-service.Dockerfile`.

Compose rendering should be checked through existing docker compose commands and
the cloud start script tests should assert the seed-disabled base env does not
need or leak `AI_GATEWAY_LOCAL_CHAT_MODEL`.
