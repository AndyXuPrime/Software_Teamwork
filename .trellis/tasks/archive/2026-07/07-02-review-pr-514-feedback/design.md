# Design

## Scope

This task is a PR takeover and review-fix task. The implementation should be conservative: keep the AI report generation fixes from PR #514, update the branch to latest `develop`, and only change code/docs needed to satisfy review feedback and current repository policy.

## Documentation Fix Design

`docs/runbooks/local-integration.md` should describe the current local runtime architecture:

```text
Docker infra -> host backend -> frontend
```

All active startup and troubleshooting instructions should route through root infrastructure Compose plus host-run scripts. Any examples that start `gateway`, `ai-gateway`, `qa`, `document`, parser, migrations, seed jobs, or frontend through Docker Compose should be removed or rewritten.

For smoke guidance, the runbook may still describe target validation intent, but the execution shape must be host-run:

- Start infra and host services through `dev-up.sh` and `run-backend.sh`.
- Inspect host-run logs through `.local/logs/<service>.log`.
- Use service ports on localhost.
- Mention external provider proxy settings through standard host environment variables in `deploy/.env`, not AI Gateway container-specific variables.

## Frontend Polling Fix Design

The current PR tries to keep polling after `job.failed` so the UI can observe asynq automatic retry transitions. The review points out that some failures are terminal, so polling every 8 seconds forever is too broad.

The safer design is bounded failed-state polling:

- Keep normal polling for non-terminal active states.
- On `failed`, continue slow polling only for a short grace window after the failed timestamp or latest failed event.
- After the grace window, stop polling and let explicit user actions or invalidation refresh state.
- If the backend exposes retry metadata in current generated types, prefer that metadata. If it does not, use a small frontend-only grace window derived from existing timestamps, because adding backend contract fields would expand the PR scope.

This preserves the intended retry recovery behavior without infinite polling for validation failures or exhausted retries.

## Compatibility

- Do not add new backend API fields unless existing types already expose retry metadata.
- Do not change public Gateway OpenAPI unless inspection proves it is already needed by PR #514.
- Preserve report outline flattening and duplicate-numbering fixes.
- Preserve Document service timeout, prompt, DOCX, and validation retry behavior unless latest `develop` already contains equivalent changes.

## Rollback

If a fix causes regressions, revert only the minimal changed hunk:

- Documentation rollback should restore the latest `develop` host-run wording, not old Compose business-service instructions.
- Frontend polling rollback should return to latest `develop` behavior or a smaller bounded polling interval, not unbounded failed polling.
