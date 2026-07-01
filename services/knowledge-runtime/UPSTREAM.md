# Vendored RAGFlow Runtime

This directory is an isolated copy of upstream RAGFlow, deployed as the Knowledge
vendor runtime. The Go contract adapter in `services/knowledge/` calls its HTTP
API on `:9380` via `VENDOR_RUNTIME_URL`.

## Upstream

- Repository: https://github.com/infiniflow/ragflow
- Branch: `main`
- Imported commit: `45fc7feab4a0da6fec2d0fecbae67fabdc9bb3a2`
- Import method: `git clone --depth 1`
- Imported on: 2026-07-01
- License: Apache License 2.0, preserved in `LICENSE`

## Isolation Boundary

- Do not import this Python source directly from the Go Knowledge adapter; call
  the documented HTTP API (`/api/v1/*` on `:9380`).
- Do not add Gateway, Parser, or QA integration inside this vendor tree.
- Product auth is Gateway `X-User-Id` / `X-Tenant-Id`; RAGFlow web login is not used.

## Refresh Notes

To refresh this copy from upstream, use a temporary clone and replace this
directory deliberately:

```bash
tmpdir=$(mktemp -d)
git clone --depth 1 https://github.com/infiniflow/ragflow.git "$tmpdir/ragflow"
rm -rf "$tmpdir/ragflow/.git"
rsync -a --delete "$tmpdir/ragflow/" services/knowledge-runtime/
rm -rf "$tmpdir"
```

After refreshing:

- update the imported commit in this file;
- keep `LICENSE` and upstream attribution files;
- verify there is no nested `.git` directory;
- review `.gitignore` interactions so the parent repository does not drop
  upstream tracked example files.
