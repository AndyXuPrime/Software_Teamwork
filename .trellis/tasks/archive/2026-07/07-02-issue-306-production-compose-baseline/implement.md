# S-031 Production Compose Baseline Implementation Plan

## Checklist

1. Inspect current service Dockerfile capabilities and env requirements.
2. Add `deploy/.env.production.example` with all required variable names and safe placeholders.
3. Add `deploy/docker-compose.production.yml`:
   - infrastructure with named persistent volumes and health checks,
   - migration jobs,
   - all landed backend services,
   - frontend placeholder service if an image is available in deployment,
   - public exposure limited to frontend and gateway by default,
   - no local demo seed services.
4. Add `deploy/production-baseline.md`:
   - purpose and difference from local/demo Compose,
   - image strategy and parser image build/pull recovery,
   - env/secrets,
   - startup order,
   - readiness checks,
   - persistence and backup/restore,
   - upgrade and rollback.
5. Update entry docs to link the new baseline and keep status language consistent.
6. Run validation:
   - `python3 scripts/check_docker_policy.py`
   - `docker compose -f deploy/docker-compose.yml --env-file deploy/.env.example config --quiet`
   - `docker compose -f deploy/docker-compose.yml --env-file deploy/.env.example --profile ai config --quiet`
   - `docker compose -f deploy/docker-compose.production.yml --env-file deploy/.env.production.example config --quiet`
   - `git diff --check`
   - focused text scans for `latest`, real-secret-looking values, local demo credentials, and conflict markers.

## Risk Points

- Production Compose config validation can pass with placeholders even though runtime startup needs real secrets; documentation must make that distinction explicit.
- Service images may not yet exist in a registry; the baseline should support image-tag replacement and optional local build commands without treating cached local images as evidence.
- Frontend Docker image may not be implemented yet. If absent, the baseline should expose a placeholder image variable and document that the gateway can remain the backend public entrypoint until frontend image publication is finalized.
- AI Gateway seeded fake profiles are local-only; production/staging must create profiles and credentials through admin/API operations after secret injection.

## Rollback Points

- If production Compose becomes too speculative, keep the new docs/env template and convert the Compose file to a config-validated staging skeleton with explicit TODOs only where service image publication is not yet implemented.
- Do not change local/demo Compose semantics unless validation shows an existing doc mismatch.
