# Vendored RAGFlow Runtime

This directory is an isolated copy of upstream RAGFlow for Knowledge/RAG
adaptation work. It is not wired into the current Knowledge service runtime.

## Upstream

- Repository: https://github.com/infiniflow/ragflow
- Branch: `main`
- Imported commit: `45fc7feab4a0da6fec2d0fecbae67fabdc9bb3a2`
- Import method: `git clone --depth 1`
- Imported on: 2026-07-01
- License: Apache License 2.0, preserved in `LICENSE`

## Isolation Boundary

- Do not import this source directly from the Go Knowledge service.
- Do not add Knowledge, Parser, Gateway, or QA integration here in the vendor
  import task.
- Future adapters should call a documented HTTP/runtime boundary or live outside
  this copied upstream tree.

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
