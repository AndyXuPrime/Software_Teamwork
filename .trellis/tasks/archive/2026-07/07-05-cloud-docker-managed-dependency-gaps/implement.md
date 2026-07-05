# Implementation Plan

1. Add Gateway Redis username/TLS fields to config, main wiring, Redis client
   options, and Gateway docs.
2. Add Gateway config and Redis option tests for username/password/db/TLS and
   invalid TLS booleans.
3. Update cloud compose and cloud env template for Gateway Redis username/TLS.
4. Update cloud compose/start/seed logic so `DOCKER_SEED_ENABLED=false` skips
   seed-only preflight and compose interpolation requirements.
5. Add a focused `scripts/docker/start.sh` unit test with fake Docker covering
   disabled seed success and enabled seed failure.
6. Update deploy/runbook/spec docs for Gateway managed Redis and disabled seed
   behavior.
7. Run validation:
   - `go test ./...` and `go build ./cmd/server` in `services/gateway`
   - targeted script unit tests
   - Docker policy/environment/local startup unit tests used by CI
   - `python3 scripts/check_docker_policy.py`
   - shell syntax checks
   - root and cloud compose config checks
   - `git diff --check`
