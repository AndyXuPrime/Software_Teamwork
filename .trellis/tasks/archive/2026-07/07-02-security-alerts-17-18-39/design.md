# Technical Design

## Scope

This task is a narrow security remediation across three surfaces:

- `services/file` dependency graph.
- `services/qa/internal/repository/postgres.go` citation metadata serialization.
- `apps/web/src/pages/qa/chat/page.tsx` client-only QA chat ID generation.
- host-run local startup defaults for Go module downloads.

It does not change public API contracts, migrations, Docker Compose service
ownership, or frontend routes.

## Alert 39: File Service Dependency

Run the File service Go module update from `services/file` so the module graph
selects `golang.org/x/net@v0.55.0` or newer. Commit both `go.mod` and `go.sum`
changes produced by Go tooling.

Compatibility expectation: this is a transitive patch-level security update
inside the Go x/* dependency family. No source changes are expected unless Go
tooling reveals incompatible transitive constraints.

## Alert 18: QA Citation Metadata Allocation

Current code allocates `map[string]any` with capacity `len(item.Metadata)+1`.
CodeQL flags the size calculation because `item.Metadata` can be influenced by
tool output JSON that originated in `agent/loop.go`.

Use a helper that starts from a nil/default-capacity map and copies metadata
keys without size arithmetic. This removes the overflow sink while preserving
semantics:

- nil or empty metadata remains serializable as `{}` unless attachment ID is
  added;
- user-provided `attachmentId` and `attachment_id` are removed;
- a trimmed non-empty `item.AttachmentID` is written as canonical
  `attachmentId`;
- all unrelated metadata keys and values are preserved.

Add repository unit tests for the helper because it is pure and already has a
local test file.

## Alert 17: QA Chat Secure IDs

Replace `Math.random()` in the page-local `nextId()` helper with Web Crypto.
Prefer `crypto.randomUUID()` when available because it is standard in modern
browsers and avoids manual byte-to-string bias. Provide a `crypto.getRandomValues`
fallback for environments without `randomUUID`, still without using
`Math.random()`.

The IDs are UI/client message IDs, not secrets, but CodeQL treats them as
security-sensitive because they enter the QA message/session flow. A secure
random source resolves the alert with minimal behavior change.

## Local Issue: Go Module Proxy Timeout

`dev-up.sh` and `run-backend.sh` already source `deploy/.env`, and the project
rules require `deploy/.env.example` to be the only default local configuration
source. Therefore the fix belongs in `deploy/.env.example`, not in script-local
fallback exports.

Add default host-run Go module settings for mainland China developer networks:

- `GOPROXY=https://goproxy.cn,direct`
- `GOSUMDB=sum.golang.google.cn`

These settings preserve checksum verification while avoiding the
`proxy.golang.org` / `sum.golang.org` path that commonly times out. Users on a
network that cannot access these endpoints can edit their copied `deploy/.env`
without changing the repository baseline.

Update local startup documentation to distinguish three independent download
paths:

- Docker image source overrides: `POSTGRES_IMAGE`, `REDIS_IMAGE`, etc.
- Python/uv package index: `UV_DEFAULT_INDEX`.
- Go module proxy/checksum database: `GOPROXY` and `GOSUMDB`.

Update the local seed/startup contract verifier so the defaults stay covered.

## Rollback

Each alert fix can be reverted independently:

- dependency update: revert `services/file/go.mod` and `go.sum`;
- QA metadata fix: revert helper and tests in `services/qa/internal/repository`;
- frontend ID fix: revert `nextId()` in `apps/web/src/pages/qa/chat/page.tsx`.
- Go proxy defaults: revert `deploy/.env.example`, local startup docs, and the
  local seed/startup contract checker changes.
