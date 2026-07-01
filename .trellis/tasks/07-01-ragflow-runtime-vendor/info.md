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
