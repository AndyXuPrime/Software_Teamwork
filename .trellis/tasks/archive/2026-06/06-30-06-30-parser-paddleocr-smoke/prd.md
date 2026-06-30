# Parser PaddleOCR smoke

## Goal

Implement issue #285 by adding an env-gated Parser PaddleOCR smoke test and
documenting the local model/runtime requirements without introducing a full
Parser HTTP runtime or making ordinary CI depend on PaddleOCR.

## Acceptance Criteria

- Normal path: `cd services/parser && PYTHONPATH=src python3 -m unittest discover -s tests -v` passes in a plain checkout and reports the real PaddleOCR smoke as skipped.
- Real runtime path: with `PARSER_PADDLEOCR_SMOKE=1` and valid PaddleOCR runtime/model configuration, the smoke loads PaddleOCR, runs OCR against a tiny fixture, and verifies non-empty text output.
- Failure path: when the smoke is explicitly enabled but runtime or model prerequisites are missing, the failure message explains which dependency or environment variable to fix.
- Forbidden states: ordinary CI must not require PaddleOCR/PaddlePaddle/model files; Knowledge and Go services must not receive PaddleOCR dependencies; Parser OpenAPI and service contract must remain unchanged.
- Documentation: Parser docs and the local runbook must describe env vars, command lines, CPU/memory expectations, default skip behavior, and troubleshooting notes for #125 follow-up reuse.

## Scope

- Add a minimal Python smoke harness under `services/parser`.
- Add a tiny local OCR fixture suitable for smoke testing.
- Update Parser service docs and local integration runbook.
- Do not create a PR; prepare a PR body draft that follows `.github/pull_request_template.md`.
