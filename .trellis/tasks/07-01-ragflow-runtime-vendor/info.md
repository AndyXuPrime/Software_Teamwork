# Implementation Notes

## Imported Runtime

- Local path: `services/knowledge/vendor/ragflow-runtime/`
- Upstream: `https://github.com/infiniflow/ragflow`
- Branch: `main`
- Commit: `45fc7feab4a0da6fec2d0fecbae67fabdc9bb3a2`
- Nested Git metadata removed: yes
- Upstream license preserved: `services/knowledge/vendor/ragflow-runtime/LICENSE`

## Boundary

This task only imports upstream source as an isolated snapshot. It does not wire
RAGFlow into Knowledge, Parser, Gateway, QA, Docker Compose, or CI.

## Verification

- Confirmed key upstream directories exist: `deepdoc`, `rag`, `api`, `web`.
- Confirmed no nested `.git` directory remains.
- Confirmed parent repository can see all 4249 files under the vendored runtime.
- Ran `git diff --check`.

## Follow-up

Future tasks can inspect the vendored source and decide which layer to adapt
first, most likely Parser-backed document parsing before retrieval replacement.

## Trim Log

### 2026-07-01: remove upstream web UI

- Removed: `services/knowledge/vendor/ragflow-runtime/web/`
- Reason: this repository already owns the frontend under `apps/web/`; the
  first RAGFlow adaptation target is backend/parser/RAG runtime behavior, not
  the upstream product UI.
- Confirmed by user before deletion: yes

### 2026-07-01: remove upstream Helm chart

- Removed: `services/knowledge/vendor/ragflow-runtime/helm/`
- Reason: this repository owns deployment wiring separately; keeping RAGFlow's
  upstream Kubernetes chart would imply an unsupported deployment path.
- Confirmed by user before deletion: yes

### 2026-07-01: remove upstream GitHub metadata

- Removed: `services/knowledge/vendor/ragflow-runtime/.github/`
- Reason: RAGFlow's upstream issue templates, PR template, CodeQL config, release
  workflow, and test workflow do not run from this vendored path and would
  confuse this repository's own root-level GitHub automation.
- Confirmed by user before deletion: yes

### 2026-07-01: remove upstream Python agent runtime

- Removed: `services/knowledge/vendor/ragflow-runtime/agent/`
- Reason: this repository's RAGFlow adaptation scope is Knowledge/RAG and
  parser/runtime behavior; agent orchestration is outside this team's boundary.
- Pre-delete note: `rag/flow/*` and `api/ragflow_server.py` still referenced the
  Python `agent` package, so later runtime work must remove or replace those
  full-RAGFlow entry points before trying to run the trimmed snapshot.
- Confirmed by user before deletion: yes

### 2026-07-01: remove upstream Python admin service and CLI

- Removed: `services/knowledge/vendor/ragflow-runtime/admin/`
- Reason: upstream RAGFlow admin service/CLI manages its own users, roles,
  service health, license, config, datasets, and agents. This repository owns
  those concerns through its own Gateway/Auth/service boundaries.
- Pre-delete note: Docker scripts, build scripts, Go admin server code, and
  heartbeat clients still reference upstream admin endpoints; later runtime work
  must remove or replace those references before trying to run the trimmed
  snapshot.
- Confirmed by user before deletion: yes

### 2026-07-01: remove tests for trimmed Web/Admin/Agent surfaces

- Removed:
  - `services/knowledge/vendor/ragflow-runtime/test/playwright/`
  - `services/knowledge/vendor/ragflow-runtime/test/testcases/test_admin_api/`
  - `services/knowledge/vendor/ragflow-runtime/test/testcases/test_web_api/test_agent_app/`
  - `services/knowledge/vendor/ragflow-runtime/test/testcases/test_sdk_api/test_agent_management/`
  - `services/knowledge/vendor/ragflow-runtime/test/unit_test/agent/`
- Reason: these tests target upstream Web UI, admin service/API, SDK agent
  management, or Python agent runtime paths that were removed from the trimmed
  snapshot.
- Kept: `test/unit_test/deepdoc/`, `test/unit_test/rag/`, `test/unit_test/mcp/`,
  and `test/fixtures/` for parser/RAG/MCP behavior reference.
- Confirmed by user before deletion: yes

### 2026-07-01: remove upstream standalone tools and plugins

- Removed: `services/knowledge/vendor/ragflow-runtime/tools/`
- Reason: upstream standalone plugins, migration helpers, install scripts, and
  developer utilities are not part of the Knowledge parser/RAG/MCP runtime
  adaptation surface.
- Pre-delete note: upstream Dockerfile and Docker launch scripts referenced
  `tools/scripts/mysql_migration.py`, so later runtime work must remove or
  replace those full-product Docker startup paths before trying to run the
  trimmed snapshot.
- Confirmed by user before deletion: yes

### 2026-07-01: remove non-core upstream Docker variants

- Removed:
  - `services/knowledge/vendor/ragflow-runtime/docker/docker-compose-CN-oc9.yml`
  - `services/knowledge/vendor/ragflow-runtime/docker/docker-compose-macos.yml`
  - `services/knowledge/vendor/ragflow-runtime/docker/oceanbase-entrypoint.sh`
  - `services/knowledge/vendor/ragflow-runtime/docker/oceanbase/`
- Reason: these are upstream deployment variants or OceanBase-specific startup
  helpers, not the likely baseline for this repository's later Knowledge/RAG
  containerization.
- Kept: `docker/docker-compose.yml`, `docker/docker-compose-base.yml`,
  `docker/entrypoint.sh`, `docker/launch_backend_service.sh`, and
  `docker/service_conf.yaml.template` as temporary reference material for later
  containerization design.
- Confirmed by user before deletion: yes

### 2026-07-01: remove upstream examples

- Removed: `services/knowledge/vendor/ragflow-runtime/example/`
- Reason: upstream chat demo, HTTP sample scripts, and SDK sample scripts are
  not runtime code and are not part of the Knowledge parser/RAG/MCP adaptation
  surface.
- Kept: `sdk/` and `docs/` for now as API/reference material until the adapter
  boundary is finalized.
- Confirmed by user before deletion: yes

### 2026-07-01: keep Chinese README and remove stale module descriptions

- Removed:
  - `services/knowledge/vendor/ragflow-runtime/README.md`
  - `services/knowledge/vendor/ragflow-runtime/README_ar.md`
  - `services/knowledge/vendor/ragflow-runtime/README_fr.md`
  - `services/knowledge/vendor/ragflow-runtime/README_id.md`
  - `services/knowledge/vendor/ragflow-runtime/README_ja.md`
  - `services/knowledge/vendor/ragflow-runtime/README_ko.md`
  - `services/knowledge/vendor/ragflow-runtime/README_pt_br.md`
  - `services/knowledge/vendor/ragflow-runtime/README_tr.md`
  - `services/knowledge/vendor/ragflow-runtime/README_tzh.md`
- Updated: `services/knowledge/vendor/ragflow-runtime/README_zh.md`
- Updated: `services/knowledge/vendor/ragflow-runtime/pyproject.toml`
- Reason: keep the upstream Chinese README as the local readable product
  overview, while removing descriptions that now point to deleted Web, Agent,
  Admin, example, and full upstream startup surfaces.
- Metadata note: root `pyproject.toml` now points `readme` to `README_zh.md`
  because root `README.md` was removed.
- Confirmed by user before deletion: yes

### 2026-07-01: remove stale upstream development metadata files

- Removed:
  - `services/knowledge/vendor/ragflow-runtime/lefthook.yml`
  - `services/knowledge/vendor/ragflow-runtime/codecov.yml`
  - `services/knowledge/vendor/ragflow-runtime/.trivyignore`
  - `services/knowledge/vendor/ragflow-runtime/.rooignore`
  - `services/knowledge/vendor/ragflow-runtime/AGENTS.md`
  - `services/knowledge/vendor/ragflow-runtime/CLAUDE.md`
  - `services/knowledge/vendor/ragflow-runtime/test.py`
- Reason: these files describe upstream local hooks, Codecov/Trivy/Roo tooling,
  assistant guidance, or a standalone FastAPI echo demo. They are not part of
  the Knowledge parser/RAG/MCP runtime adaptation surface and several still
  reference removed Web, Agent, Admin, tools, or full-product startup paths.
- Kept: `services/knowledge/vendor/ragflow-runtime/show_env.sh` at user request.
- Confirmed by user before deletion: yes

### 2026-07-01: keep and harden upstream Go test helper

- Updated: `services/knowledge/vendor/ragflow-runtime/run_go_tests.sh`
- Reason: keep a lightweight upstream Go-side regression entry point for
  retained Go packages, while removing the missing `./internal/cache/...`
  package entry and skipping future missing package directories after trimming.
- Validation note: `go list` over the selected package patterns was attempted
  but did not return within 90 seconds, likely due to upstream module dependency
  resolution, so it was interrupted. The script update is validated with shell
  syntax and targeted diff checks rather than a full Go test run.
- Confirmed by user before edit: yes

### 2026-07-01: remove non-core upstream product documentation

- Removed:
  - `services/knowledge/vendor/ragflow-runtime/Dockerfile.scratch.oc9`
  - `services/knowledge/vendor/ragflow-runtime/SECURITY.md`
  - `services/knowledge/vendor/ragflow-runtime/check_comment_ascii.py`
  - `services/knowledge/vendor/ragflow-runtime/deepdoc/README_tr.md`
  - `services/knowledge/vendor/ragflow-runtime/docs/develop/deepwiki.md`
  - `services/knowledge/vendor/ragflow-runtime/docs/develop/contributing.md`
  - `services/knowledge/vendor/ragflow-runtime/docs/administrator/admin/`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/team/`
  - Docusaurus-only `docs/**/_category_.json` files, except
    `docs/develop/mcp/_category_.json`
  - Selected Agent/UI/product docs unrelated to retained Knowledge/RAG/MCP
    reference, including Agent introduction, embed, ecommerce quickstart,
    sandbox quickstart, Agent acceleration, Deep Research, Memory UI, and Agent
    Context Engine overview.
- Reason: these files describe upstream security/contribution processes,
  Docusaurus navigation, localized or UI-only docs, upstream Admin/team/Agent
  product flows, or special OpenCloudOS full-product build paths. They are not
  part of the retained document parsing, RAG, MCP/tooling, or containerization
  reference surface.
- Kept: MCP docs, RAG/dataset/parser/chunker/indexer/retrieval/transformer
  reference docs, DeepDoc English/Chinese READMEs, and Docker/containerization
  reference files.
- Confirmed by user before deletion: yes

### 2026-07-01: remove upstream product operations and chat docs

- Removed:
  - `services/knowledge/vendor/ragflow-runtime/docs/administrator/tracing.mdx`
  - `services/knowledge/vendor/ragflow-runtime/docs/administrator/upgrade_ragflow.mdx`
  - `services/knowledge/vendor/ragflow-runtime/docs/administrator/configurations/`
  - `services/knowledge/vendor/ragflow-runtime/docs/administrator/migration/`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/chat/`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/ai_search.md`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/manage_files.md`
  - `services/knowledge/vendor/ragflow-runtime/docs/quickstart.mdx`
  - `services/knowledge/vendor/ragflow-runtime/docs/faq.mdx`
  - `services/knowledge/vendor/ragflow-runtime/docs/release_notes.md`
- Updated:
  - `services/knowledge/vendor/ragflow-runtime/README_zh.md`
  - `services/knowledge/vendor/ragflow-runtime/docs/references/glossary.mdx`
- Reason: these files describe upstream full-product deployment, upgrade,
  tracing, chat UI, AI Search, file manager, FAQ, and release-note surfaces.
  They are not core parsing/RAG/MCP/tooling references for the Knowledge
  service adaptation, and several refer to deleted Web/Admin/Agent surfaces.
- Kept: API key instructions because retained MCP docs still reference them.
- Confirmed by user before deletion: yes

### 2026-07-01: remove upstream developer launch docs and dataset auxiliaries

- Removed:
  - `services/knowledge/vendor/ragflow-runtime/docs/develop/agent-go-port-design.md`
  - `services/knowledge/vendor/ragflow-runtime/docs/develop/build_docker_image.mdx`
  - `services/knowledge/vendor/ragflow-runtime/docs/develop/launch_ragflow_from_source.md`
  - `services/knowledge/vendor/ragflow-runtime/docs/develop/switch_doc_engine.md`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/dataset/add_data_source/`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/dataset/advanced/auto_metadata.md`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/dataset/advanced/autokeyword_autoquestion.mdx`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/dataset/advanced/enable_raptor.md`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/dataset/advanced/extract_table_of_contents.md`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/dataset/best_practices/accelerate_doc_indexing.mdx`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/dataset/configure_child_chunking_strategy.md`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/dataset/enable_excel2html.md`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/dataset/manage_metadata.md`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/dataset/set_context_window.md`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/dataset/set_metadata.md`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/dataset/set_page_rank.md`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/dataset/use_tag_sets.md`
- Reason: these pages cover full-product source launching, Docker image build,
  engine switching, third-party data source connectors, and advanced dataset
  tuning options that are outside the retained parser/RAG/MCP reference set
  for the vendor snapshot.
- Kept: `configure_knowledge_base.md`, `run_retrieval_test.md`,
  `select_pdf_parser.md`, and `construct_knowledge_graph.md` for core
  ingestion and retrieval reference.
- Confirmed by user before deletion: yes

### 2026-07-01: remove upstream agent component side docs and SDK examples

- Removed:
  - `services/knowledge/vendor/ragflow-runtime/.agents/`
  - `services/knowledge/vendor/ragflow-runtime/sdk/python/hello_ragflow.py`
  - `services/knowledge/vendor/ragflow-runtime/sdk/python/test.py`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/agent/agent_component_reference/await_response.mdx`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/agent/agent_component_reference/categorize.mdx`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/agent/agent_component_reference/code.mdx`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/agent/agent_component_reference/execute_sql.md`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/agent/agent_component_reference/http.md`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/agent/agent_component_reference/iteration.mdx`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/agent/agent_component_reference/message.md`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/agent/agent_component_reference/switch.mdx`
  - `services/knowledge/vendor/ragflow-runtime/docs/guides/agent/agent_component_reference/text_processing.mdx`
- Reason: these are upstream assistant-internal files, agent component subpages
  beyond the parser/RAG/MCP-adjacent subset, or SDK demo scripts. They are not
  needed for the isolated runtime import.
- Kept: `agent.mdx`, `begin.md`, `chunker_title.md`, `chunker_token.md`,
  `indexer.md`, `parser.md`, `retrieval.mdx`, and `transformer.md`.
- Confirmed by user before deletion: yes

### 2026-07-01: remove upstream HTTP benchmark CLI

- Removed: `services/knowledge/vendor/ragflow-runtime/test/benchmark/`
- Reason: this benchmark targets a running full RAGFlow server and includes
  both retrieval and Chat Assistant/OpenAI-compatible chat flows. The retained
  design target is document parsing, RAG retrieval, containerization, and MCP
  exposure; Chat Assistant and agent-facing product flows are outside this
  service boundary.
- Follow-up note: if benchmark coverage is needed later, add a small local
  benchmark against this repository's Knowledge/RAG/MCP APIs instead of keeping
  the upstream full-product benchmark.
- Confirmed by user before deletion: yes

### 2026-07-01: remove orphan product APIs, benchmarks, and connector dependencies

- Removed:
  - upstream full-product HTTP/SDK/Web/Admin/Agent/chat/session/search/memory
    tests that target already-trimmed surfaces
  - orphan Python Web/CLI helper modules such as
    `api/apps/llm_app.py`, `api/apps/restful_apis/_generation_params.py`,
    `api/settings.py`, `api/validation.py`, `api/common/exceptions.py`,
    `api/utils/base64_image.py`, `api/utils/crypt.py`, and
    `api/utils/log_utils.py`
  - unused Go product residues such as `internal/handler/llm.go`,
    `internal/service/llm.go`, and `internal/service/deep_researcher.go`
  - `services/knowledge/vendor/ragflow-runtime/docs/develop/mcp/_category_.json`
    because it is only Docusaurus navigation metadata, not MCP runtime docs
- Updated:
  - `services/knowledge/vendor/ragflow-runtime/README_zh.md` to remove the
    stale update entry for Confluence/Notion/Discord/Google Drive sync after
    connector ingestion was trimmed
  - `services/knowledge/vendor/ragflow-runtime/pyproject.toml` and `uv.lock`
    to drop dependencies that no retained source imports, including upstream
    connector/channel/search-product packages (`akshare`, `arxiv`,
    `atlassian-python-api`, `browser-use`, `boxsdk`, `dropbox`,
    `duckduckgo-search`, `feedparser`, `google-api-python-client`,
    `google-cloud-bigquery`, `google-search-results`, `jira`, `moodlepy`,
    `Office365-REST-Python-Client`, `webdav4`, `wikipedia`, `yfinance`,
    `pyairtable`, `pygithub`, `asana`, `python-gitlab`,
    `alibabacloud-dingtalk`, `lark-oapi`, `discord.py`,
    `python-telegram-bot`, `line-bot-sdk`, `wechatpy`, `slack-sdk`,
    `deepl`, `bio`, and `pyodbc`) plus stale upstream UI/CI test packages
    (`pytest-playwright`, `codecov`)
- Kept:
  - DeepDoc parser runtime and parser docs
  - RAG retrieval, chunking, GraphRAG, model provider, embedding, rerank, and
    storage backend code
  - MCP server/client code and MCP docs
  - Docker/containerization reference files including DeepDoc and TEI images
  - URL/web-document ingestion dependencies such as `crawl4ai`, Selenium, and
    `webdriver-manager`
- Validation:
  - `uv lock` completed and removed stale direct/transitive packages from
    `uv.lock`
  - dependency/import residue searches found no retained source imports for the
    removed dependency set
  - `python3 -m py_compile api/apps/__init__.py api/db/db_models.py
    common/settings.py test/testcases/conftest.py test/unit_test/conftest.py`
    passed
  - `bash -n docker/entrypoint.sh && bash -n docker/launch_backend_service.sh
    && bash -n build.sh && bash -n run_go_tests.sh` passed
  - `gofmt` was run over modified retained Go files
  - `git diff --check` passed after trimming trailing whitespace and redundant
    EOF blank lines in touched vendored files
  - `timeout 120s go test ./internal/common ./internal/utility
    ./internal/router ./internal/server ./internal/handler ./internal/service`
    did not complete before timeout while downloading/compiling upstream Go
    dependencies; no test failure was observed before timeout
- Confirmed by user before deletion: yes, as a combined non-core cleanup batch
  after the benchmark deletion was approved

### 2026-07-01: remove agent planning and memory prompt residue

- Removed:
  - `services/knowledge/vendor/ragflow-runtime/rag/prompts/analyze_task_system.md`
  - `services/knowledge/vendor/ragflow-runtime/rag/prompts/analyze_task_user.md`
  - `services/knowledge/vendor/ragflow-runtime/rag/prompts/next_step.md`
  - `services/knowledge/vendor/ragflow-runtime/rag/prompts/reflect.md`
  - `services/knowledge/vendor/ragflow-runtime/rag/prompts/summary4memory.md`
  - `services/knowledge/vendor/ragflow-runtime/rag/prompts/rank_memory.md`
  - `services/knowledge/vendor/ragflow-runtime/rag/prompts/tool_call_summary.md`
  - `services/knowledge/vendor/ragflow-runtime/rag/prompts/ask_summary.md`
- Updated:
  - `services/knowledge/vendor/ragflow-runtime/rag/prompts/generator.py`
- Reason: these prompt files and helper functions support RAGFlow's own agent
  planning, next-step selection, reflection, tool-call summarization, and
  memory ranking. Agent orchestration is outside this team's Knowledge/RAG
  runtime boundary; retained MCP/tool exposure should remain as external
  capability rather than RAGFlow-internal agent planning.
- Removed from `generator.py`:
  - `memory_prompt`
  - `tool_schema`
  - `form_history`
  - `analyze_task_async`
  - `next_step_async`
  - `reflect_async`
  - `tool_call_summary`
  - `rank_memories_async`
  - related prompt loads and unused constants/imports
- Kept:
  - retrieval/citation/query rewrite prompts
  - metadata filter generation
  - TOC extraction/relevance prompts
  - image/figure description prompts
  - MCP server/client code and MCP docs
- Validation:
  - exact residue scan found no retained references to the removed functions,
    prompt constants, or `load_prompt(...)` names
  - `python3 -m py_compile rag/prompts/generator.py rag/prompts/template.py`
    passed
  - `git diff --check` passed
- Confirmed by user before deletion: yes

### 2026-07-01: restore connector source as multi-source ingestion reference

- Restored:
  - `services/knowledge/vendor/ragflow-runtime/common/data_source/`
- Added:
  - `services/knowledge/vendor/ragflow-runtime/common/data_source/README.md`
- Reason: the user wants the Knowledge/RAG system to keep third-party data
  source connector source as reference and adapter candidates, so the future
  product can visibly support multi-source knowledge ingestion.
- Boundary:
  - connector source is retained as upstream reference code only
  - connector HTTP API and DB service wiring are not restored
  - connector-specific dependencies are not re-added to default
    `pyproject.toml`
  - connector tests are not restored into the default test surface
- Follow-up note: future production integration should introduce explicit
  Knowledge/File/Auth/job contracts and optional dependency groups before
  enabling any connector family at runtime.
- Confirmed by user before restoration: yes

### 2026-07-01: remove connector runtime residue outside reference source

- Kept:
  - `services/knowledge/vendor/ragflow-runtime/common/data_source/` as
    reference source for future multi-source ingestion adapters.
- Removed or kept removed:
  - connector HTTP API and Python DB service wiring
  - Go connector DAO/entity/handler/service product wiring
  - connector sync server entry points
  - connector HTTP/Web/unit tests outside the retained source reference
  - connector product docs and default dependency wiring
- Updated:
  - dataset update validation no longer accepts the upstream `connectors`
    request field
  - dataset REST tests no longer assert a `connectors` response field
  - active RAG image-fetch comments no longer reference the connector source
    tree
- Reason: connector implementations should remain visible as candidate adapter
  code, while the trimmed runtime should not advertise or expose upstream
  connector behavior before this repository defines Knowledge/File/Auth/job
  contracts.
- Confirmed by user before cleanup: yes

### 2026-07-01: clean local artifacts and stale vendor ignore rules

- Removed from the working tree:
  - Python `__pycache__/` and `*.pyc` files generated during local syntax
    checks under `services/knowledge/vendor/ragflow-runtime/`.
- Updated:
  - `services/knowledge/vendor/ragflow-runtime/.gitignore`
- Reason: local Python bytecode caches are not source, tests, MCP behavior, or
  containerization material. The vendored `.gitignore` still described removed
  upstream Web, SDK, frontend, OceanBase, SeekDB, and Docusaurus-era paths, so
  it was reduced to rules that still match the retained parser/RAG/MCP/container
  reference surface and local build/runtime artifacts.
- Kept:
  - parser/RAG/MCP/containerization source and tests
  - model/dependency cache ignores used by retained Docker references
  - Python virtualenv, coverage, log, C++ build, Go build, and local agent state
    ignores
- Confirmed by user before cleanup: yes

### 2026-07-01: remove Python Web/Admin startup residue

- Updated:
  - `services/knowledge/vendor/ragflow-runtime/api/ragflow_server.py`
  - `services/knowledge/vendor/ragflow-runtime/api/db/init_data.py`
  - `services/knowledge/vendor/ragflow-runtime/docker/entrypoint.sh`
  - `services/knowledge/vendor/ragflow-runtime/docker/launch_backend_service.sh`
- Removed:
  - Python-side `--init-superuser` command-line option
  - `init_superuser()` and its default admin credential initialization path
  - stale `init_web_db` / `init_web_data` naming in the retained Python and
    Docker startup chain
- Renamed:
  - `init_web_data()` to `init_runtime_data()`
- Kept:
  - database table initialization through `init_database_tables()`
  - system settings initialization
  - document count repair during runtime data initialization
  - Docker startup for the Python server, task executor workers, and MCP server
- Reason: the retained Knowledge runtime should keep parser/RAG/MCP/container
  startup behavior, but should not keep upstream Web/Admin superuser bootstrap
  semantics after Web/Admin product surfaces were trimmed.
- Confirmed by user before cleanup: yes

### 2026-07-01: remove Go Admin server and license-gate residue

- Removed:
  - `services/knowledge/vendor/ragflow-runtime/internal/service/admin_client.go`
  - `services/knowledge/vendor/ragflow-runtime/internal/server/local/admin_status.go`
  - Go-side Admin heartbeat client initialization from `cmd/ingestor.go`
  - commented-out Admin gRPC control, heartbeat, reconnect, and task-assignment
    code from `internal/ingestion/ingestion_service.go`
  - `AdminConfig`, `DefaultSuperUser`, `GetAdminConfig()`, and
    `DEFAULT_SUPERUSER_*` parsing from `internal/server/config.go`
  - Admin availability / license-gate checks from retained auth and user login
    handlers
  - Admin/default-superuser environment display from system settings
  - Admin server YAML blocks, Admin command examples, and Admin-only Docker
    port variables from retained configuration references
- Kept:
  - ordinary user, tenant, auth-token, and API-token data structures
  - retained dataset/document/MCP API permission context
  - parser, ingestion, RAG/retrieval, MCP, and container startup surfaces
  - task executor heartbeat inspection, which is runtime observability rather
    than upstream Admin server heartbeat
- Reason: this Knowledge/RAG runtime should not expose or depend on upstream
  Admin server, commercial license status, heartbeat reporting, or default
  superuser bootstrap semantics. The retained runtime still needs normal user
  and tenant context for dataset/document/API-token permission behavior.
- Confirmed by user before cleanup: yes

### 2026-07-01: remove orphan product integrations and upstream identity surfaces (batches A–D)

- Removed:
  - Chat/dialog residue: `think_tag.go`, `internal/observability/otel/`
  - Dify retrieval integration (Python API, Go handler, router routes, tests)
  - Go user registration/login/profile routes (`internal/handler/user.go`) and tenant
    member/invite/list product routes from `tenant.go` / `router.go`
  - Python `user_account_service.py`, OceanBase status endpoint and dedicated tests
  - `RAGFlowWebApiAuth` test helper and `--client-type web` option
- Updated:
  - `README_zh.md`, `run_retrieval_test.md`, `glossary.mdx` to local trimmed-runtime docs
  - Docker entrypoint/help text: API server terminology (`--disable-api-server`)
  - `docker/.env` comments to drop web console / agent upload wording
  - stale dialog/Dify/chat-assistant comments in `system_api.py`, `run_tests.py`,
    `chat_model.py`, `es_conn_base.py`
  - Go user serialization to stop exposing `is_superuser`
- Kept:
  - API token / JWT auth middleware (`internal/handler/auth.go`) for core dataset,
    document, retrieval, MCP, and provider routes
  - tenant internal store APIs and default-model configuration handlers
  - OceanBase/SeekDB engine code paths (only removed product monitoring endpoint/tests)
- Confirmed by user before cleanup: yes

### 2026-07-01: restore chunk feedback retrieval weighting

- Restored: `api/db/services/chunk_feedback_service.py`
- Added: `test/unit_test/api/db/services/test_chunk_feedback_service.py`
- Reason: user feedback on cited chunks (thumb up/down) adjusts `pagerank_fea` to
  improve future retrieval quality; treated as a core RAG enhancement rather
  than upstream Chat UI residue. Feature remains opt-in via
  `CHUNK_FEEDBACK_ENABLED=true`.
- Confirmed by user before restoration: yes

### 2026-07-01: add chunk feedback REST endpoint

- Added: `api/apps/restful_apis/chunk_feedback_api.py` (`POST /api/v1/chunk-feedback`)
- Added: `test/testcases/restful_api/test_chunk_feedback_routes_unit.py`
- Updated: `docs/references/http_api_reference.md`, `README_zh.md`
- Contract: `{ "thumbup": bool, "reference": { "chunks": [...] } }` using chunks from
  retrieval/search responses; requires `CHUNK_FEEDBACK_ENABLED=true` for weight updates.

### 2026-07-01: remove orphan chat/agent code and Go product surfaces (batches E+F+G partial)

- Removed (batch E):
  - broken `GET /system/stats` route (handler missing) and its REST test
  - `internal/entity/models/llm.go` (`EinoChatModel` / `cloudwego/eino`)
  - chat dialog query rewrite: `full_question` (Python/Go), `full_question_prompt.md`
  - unused `GetChatModelConfig` / `isImage2TextLLM` in Go model service
- Removed (batch F):
  - Go `/system/variables`, `/system/environments`, duplicate `/system/keys` routes
  - `/authors/:author_id/documents` route and document service handler method
  - dead user registration/login/profile/password methods from `internal/service/user.go`
    (kept JWT/API-token/beta-token auth paths)
- Removed (batch G partial):
  - legacy Go `/v1/kb/*` routes and `internal/handler/kb.go`
  - `docs/guides/models/deploy_local_llm.mdx` (Docusaurus-only local LLM guide)
  - manual GraphRAG dev scripts `rag/graphrag/*/smoke.py`
- Updated:
  - `docs/guides/models/llm_api_key_setup.md` to drop link to removed deploy guide
  - `go.mod` to drop direct `cloudwego/eino` dependency
- Kept:
  - `/system/tokens`, status/health/config/log routes
  - `/files`, `/chat/completions`, MCP, dataset/document/retrieval APIs
  - `CrossLanguages` / `KeywordExtraction` retrieval helpers
- Validation:
  - `gofmt` over touched Go files
  - `python3 -m py_compile rag/prompts/generator.py test/testcases/restful_api/test_system.py`
  - `bash -n run_go_tests.sh`
  - `go build ./internal/router/... ./internal/handler/... ./internal/service/...`
- Confirmed by user before cleanup: yes (E+F full; G: kb legacy + deploy_local_llm + smoke only)

### 2026-07-01: remove vendor JWT/API-token auth layer (batch I)

- Removed:
  - Go `internal/handler/api_token.go`, `internal/service/api_token.go`,
    `internal/dao/api_token.go` and related beta tests
  - Go `/system/tokens` routes and Python `system_api` token management endpoints
  - JWT / API token / beta token resolution from `internal/handler/auth.go`
  - `GetUserByToken` / `GetUserByAPIToken` / `GetUserByBetaAPIToken` from user service
- Updated:
  - Go and Python auth middleware trust upstream Gateway headers
    `X-Tenant-Id` (fallback `X-User-Id`) and resolve tenant via `GetUserByTenantID`
  - `README_zh.md` auth boundary description
  - REST integration helpers: `RestClient` sends `X-Tenant-Id`; `RAGFLOW_TENANT_ID` env
  - `test_system.py`, `test_retrieval.py` gateway auth contract tests
- Restored:
  - `jsonError` helper in `internal/handler/error.go` (accidentally removed with
    legacy `kb.go` in batch G)
- Kept:
  - `api_token` DB table model and `APITokenService` (schema only; no HTTP surface)
  - Go `entity.APIToken` in auto-migrate for existing deployments
  - superuser forbidden guard in Go gateway auth
- Reason: this repository's Gateway owns identity; the vendored runtime should not
  expose upstream product auth (login, JWT issuance, API token self-service).
- Confirmed by user before cleanup: yes

### 2026-07-01: retain file manager and model chat proxy surfaces

- Kept intentionally (not in future trim scope unless product boundary changes):
  - `/files` (Go + Python `file_api.py`): upstream personal file-cabinet product flow
    (upload/folder/move → link-to-datasets), distinct from raw object storage
  - `POST /chat/completions`: OpenAI-compatible chat inference proxy over tenant
    model providers (complements `/providers` / `/models` configuration routes)
- Reason: user confirmed these remain as reference/adaptation candidates for
  file-ingestion UX and third-party/local LLM invocation, even though core
  parser/RAG retrieval does not require them at HTTP layer.
- Confirmed by user: yes

### 2026-07-01: knowledge vendor replacement phase 1 scaffold

- Added contract adapter scaffold:
  - `services/knowledge/cmd/adapter/`
  - `services/knowledge/internal/adapter/`
  - `services/knowledge/internal/adapterconfig/`
- Added runtime deployment scaffolding:
  - `services/knowledge/runtime/entrypoint.sh` (`KNOWLEDGE_RUNTIME_MODE=legacy|adapter`)
  - `services/knowledge/runtime/service_conf.compose.yaml`
  - `services/knowledge/runtime/README.md`
- Updated `services/knowledge/Dockerfile` to build both legacy server and adapter
- Updated `deploy/docker-compose.yml`:
  - optional `elasticsearch` + `knowledge-minio-init` under profile `knowledge-v2`
  - `KNOWLEDGE_RUNTIME_MODE` / `VENDOR_RUNTIME_URL` env on `knowledge` service
- Default compose path unchanged (`legacy` mode); adapter mode is opt-in
- Next: Phase 2 PostgreSQL metadata port for vendor ORM

### 2026-07-01: knowledge vendor replacement phase 2 PostgreSQL metadata port

- Go vendor database layer:
  - `internal/dao/database_dialect.go` — postgres/mysql DSN + benign migration errors
  - `internal/dao/migration_postgres.go` — PG manual migrations (tenant_llm PK, user.email unique)
  - `internal/server/config.go` — `DB_TYPE=postgres`, `postgres:` YAML, `DATABASE_URL` override
  - entity GORM tags: `type:longtext` → `type:text`
  - `internal/dao/user_tenant.go` — `TO_CHAR` on PostgreSQL
- Runtime config:
  - `services/knowledge/runtime/service_conf.compose.yaml` now targets `knowledge_system` postgres
- Python path: existing `DB_TYPE=postgres` Peewee support reused (no code changes required)
- Coexistence: legacy goose tables remain; vendor AutoMigrate creates separate RAGFlow tables
- Next: Phase 3 contract adapter routes

### 2026-07-01: knowledge vendor replacement phase 3 contract adapter routes

- Added `internal/vendorclient/` HTTP client for vendor `:9380` with gateway header forwarding
- Implemented adapter contract routes:
  - knowledge base CRUD → vendor `/api/v1/datasets`
  - document list/upload/get/delete/chunks/content → vendor datasets/documents APIs
  - knowledge query → vendor `/api/v1/datasets/search`
  - gateway RBAC via `X-User-Id` / `X-User-Roles` / `X-User-Permissions`
  - standard `{data, requestId}` / `{error}` envelopes aligned with legacy HTTP layer
- Parser-config admin routes registered but return not-implemented in adapter mode (Phase 3b)
- Next: integration tests against running vendor runtime, parser-config bridge

### 2026-07-01: knowledge vendor replacement phase 3b parser-config bridge

- Adapter wires legacy goose `parser_configs` via optional `DATABASE_URL`
- `cmd/adapter` connects PostgreSQL and injects `service.Service` for parser-config CRUD only
- Without `DATABASE_URL`, parser-config routes return `502 dependency_error`
- Vendor-facing KB/document/query routes unchanged (still proxied through vendorclient)
