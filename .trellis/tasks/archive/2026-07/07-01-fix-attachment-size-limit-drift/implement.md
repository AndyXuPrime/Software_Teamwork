# Implementation Plan: Attachment size limit alignment

1. Add a 20 MiB QA config ceiling and regression tests.
2. Extend `FileHTTPConfig` with a maximum read size, wire it from `cmd/server`, and reject over-limit response bodies explicitly.
3. Add attachment client tests for exact-limit and over-limit reads.
4. Run focused and full QA/Gateway verification, then inspect the PR diff against both findings.
