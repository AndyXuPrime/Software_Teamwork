# Vendor isolated RAGFlow runtime for Knowledge

## Goal

Bring the upstream RAGFlow project into this repository as an isolated runtime
under the Knowledge service area so later tasks can inspect, trim, and adapt
document parsing and RAG capabilities without mixing upstream source directly
into the existing Go Knowledge service.

## What I already know

- The user wants to first pull the complete RAGFlow project locally, then
  gradually adjust and delete unneeded pieces.
- The upstream project is `https://github.com/infiniflow/ragflow`.
- Existing project boundaries keep Knowledge as the owner of ingestion state,
  chunking, embedding, indexing, and retrieval, while parser runtimes stay
  behind HTTP contracts.
- This task is only the vendor/import step. It must not wire RAGFlow into
  Knowledge, Parser, Gateway, or QA yet.

## Assumptions

- Store RAGFlow under `services/knowledge/vendor/ragflow-runtime/` so it is
  clearly attached to Knowledge work but remains isolated from current Go
  packages.
- Preserve upstream history metadata in a small repo-local note because this
  repository cannot nest a live Git repository cleanly inside normal source
  tracking.
- Keep upstream license and attribution files.

## Requirements

- Pull the complete upstream RAGFlow source into an isolated directory.
- Do not modify existing Knowledge service behavior.
- Do not add imports from the current Go services into RAGFlow or from RAGFlow
  into current Go services.
- Record upstream URL, branch, commit, and local refresh instructions.
- Make the isolation boundary obvious to future contributors.

## Acceptance Criteria

- [x] `services/knowledge/vendor/ragflow-runtime/` contains the RAGFlow source.
- [x] The vendored directory does not contain a nested `.git` directory.
- [x] A local note records upstream URL, branch/ref, commit, license, and refresh
      instructions.
- [x] `git status` clearly shows the vendored import and task files only.
- [x] No existing service code is changed for runtime integration.

## Definition of Done

- Task exists and is started.
- RAGFlow is pulled into the isolated directory.
- Basic source presence is verified.
- No build/test suite is run unless needed, because this is a source import only.

## Out of Scope

- Parser API adapter.
- Knowledge retrieval adapter.
- Docker Compose integration.
- Runtime dependency installation.
- RAGFlow code trimming or behavior modification.
- Agent/chat/workflow integration.

## Technical Notes

- Current Knowledge service lives under `services/knowledge/`.
- Existing repo guidance treats `services/parser/` as the runtime boundary for
  Python/OCR parsing dependencies.
- Later implementation tasks should decide whether RAGFlow backs only parser
  behavior or a fuller retrieval engine.
