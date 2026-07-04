#!/usr/bin/env python3
"""Verify the root local/demo seed contract for integration fixtures."""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path


SEED_001 = Path("deploy/seeds/001-local-demo-seed.sql")
SEED_002 = Path("deploy/seeds/002-ai-gateway-model-profiles.sql")
SEED_003 = Path("deploy/seeds/003-qa-document-mcp.sql")
SEED_004 = Path("deploy/seeds/004-qa-default-knowledge-base.sql")
CLEANUP_SEED = Path("deploy/seeds/099-local-demo-cleanup.sql")
DEPLOY_README = Path("deploy/README.md")
LOCAL_RUNBOOK = Path("docs/runbooks/local-integration.md")
ENV_EXAMPLE = Path(".env.example")
CONFIG_README = Path("config/README.md")
CONFIG_BASE = Path("config/base.yaml")
GITIGNORE = Path(".gitignore")
AUTH_MIGRATIONS_DIR = Path("services/auth/migrations")
CHECK_SCRIPT = Path("scripts/local/check.sh")
START_SCRIPT = Path("scripts/local/start.sh")
CLEAN_SCRIPT = Path("scripts/local/clean.sh")
AI_GATEWAY_LOCAL_SEED_RENDERER = Path("scripts/local/render_ai_gateway_local_seed.go")
STOP_SCRIPT = Path("scripts/local/stop.sh")
LOCAL_COMMON_HELPER = Path("scripts/local/lib/common.sh")
LOCAL_PROCESS_HELPER = Path("scripts/local/lib/process.sh")
LOCAL_KNOWLEDGE_RUNTIME_HELPER = Path("scripts/local/lib/knowledge-runtime.sh")
AI_GATEWAY_LOCAL_SEED_MAIN = Path("services/ai-gateway/cmd/local-seed/main.go")
PUBLIC_LOCAL_ENTRYPOINTS = [
    CHECK_SCRIPT,
    START_SCRIPT,
    STOP_SCRIPT,
    CLEAN_SCRIPT,
]

REQUIRED_SEED_001_TOKENS = {
    "Auth local admin user": ["usr_local_admin", "cred_local_admin_password", "urole_local_admin_admin"],
    "Auth local super admin user": [
        "usr_local_super_admin",
        "cred_local_super_admin_password",
        "urole_local_super_admin_super_admin",
    ],
    "Knowledge sample": ["kb_local_demo", "doc_local_demo_seed", "chunk_local_demo_seed_001"],
    "Document sample": [
        "22222222-2222-4222-8222-222222222201",
        "22222222-2222-4222-8222-222222222301",
        "22222222-2222-4222-8222-222222222401",
        "22222222-2222-4222-8222-222222222501",
        "22222222-2222-4222-8222-222222222502",
        "22222222-2222-4222-8222-222222222601",
        "22222222-2222-4222-8222-222222222602",
    ],
    "QA sample": [
        "33333333-3333-4333-8333-333333333301",
        "33333333-3333-4333-8333-333333333401",
        "33333333-3333-4333-8333-333333333402",
        "33333333-3333-4333-8333-333333333501",
        "33333333-3333-4333-8333-333333333502",
    ],
}

REQUIRED_DATABASE_SECTIONS = [
    r"\\connect\s+auth_system",
    r"\\connect\s+knowledge_system",
    r"\\connect\s+document_system",
    r"\\connect\s+qa_system",
]

REQUIRED_AI_TOKENS = [
    "default-chat",
    "default-embedding",
    "default-rerank",
    "http://localhost:11434/v1",
    "cred-local-chat",
    "cred-local-embedding",
    "cred-local-rerank",
    "local-demo-key-v1",
]

REQUIRED_DOCUMENT_MCP_TOKENS = [
    r"\\connect\s+qa_system",
    "33333333-3333-4333-8333-333333333601",
    "'document'",
    "'Document MCP'",
    "'streamable_http'",
    "'http://localhost:8085/mcp'",
    "'Authorization'",
    "'local-seed'",
    "ON CONFLICT (alias) DO UPDATE",
]

REQUIRED_QA_DEFAULT_KB_TOKENS = [
    r"\\connect\s+qa_system",
    "keep QA's default knowledge-base list empty",
    "defaultKnowledgeBaseIds",
    "search all indexed",
    "qa_config_knowledge_bases",
    "kb_local_demo",
    "DELETE FROM qa_config_knowledge_bases",
]

FORBIDDEN_AI_TOKENS = [
    "host.docker.internal",
]

REQUIRED_AUTH_MIGRATION_TOKENS = {
    "QA settings permission": [
        "perm_qa_settings_read",
        "qa:settings:read",
        "perm_qa_settings_write",
        "qa:settings:write",
    ],
    "admin QA settings grant": [
        "rperm_admin_qa_settings_read",
        "rperm_admin_qa_settings_write",
        "'admin', 'qa:settings:read'",
        "'admin', 'qa:settings:write'",
    ],
    "super admin QA settings grant": [
        "rperm_super_qa_settings_read",
        "rperm_super_qa_settings_write",
        "'super_admin', 'qa:settings:read'",
        "'super_admin', 'qa:settings:write'",
    ],
}

REQUIRED_DOC_TOKENS = [
    "configuration authority",
    "config/base.yaml",
    "config/dev.yaml",
    ".env.local",
    "LOCAL_ADMIN_USERNAME=admin",
    "LOCAL_ADMIN_PASSWORD=LocalDemoAdmin#12345",
    "LOCAL_SUPER_ADMIN_USERNAME=superadmin",
    "LOCAL_SUPER_ADMIN_PASSWORD=LocalDemoAdmin#12345",
    "admin / LocalDemoAdmin#12345",
    "superadmin / LocalDemoAdmin#12345",
    "cp .env.example .env.local",
    "Go modules 下载只在",
    "源选择采用官方默认源",
    "active 第三方镜像值",
    "默认使用官方源",
    "--china",
    "大陆镜像",
    "GOPROXY=https://proxy.golang.org,direct",
    "GOSUMDB=sum.golang.org",
    "./scripts/local/check.sh",
    "./scripts/local/start.sh",
    "./scripts/local/stop.sh",
    "down -v",
    "AI_GATEWAY_LOCAL_SEED_ENABLED=true",
    "AI_GATEWAY_LOCAL_PROVIDER_API_KEY=<local-provider-api-key>",
    "default-chat",
]

REQUIRED_CHECK_TOKENS = [
    "[check]",
    "no downloads or builds will run",
    "setup suggestions",
    "Official sources, run manually only for missing items",
    "Mainland China mirrors, run manually only for missing items",
    "--sync-only --profile",
    "ragflow_deps/download_deps.py --skip-uv-sync",
    "docker.1ms.run/library/postgres:16-alpine",
    "goose@v3.27.1",
    "https://go.dev/dl/",
    "https://docs.astral.sh/uv",
]

FORBIDDEN_CHECK_TOKENS = [
    "docker image inspect",
    "docker_image_present",
    "check_docker_images",
]

REQUIRED_START_TOKENS = [
    "[start]",
    "This script does not run dependency downloads",
    "--pull never",
    ".local/tools/config-ctl",
    ".local/tools/goose",
    ".local/bin",
    "AUTH_DATABASE_URL",
    "FILE_DATABASE_URL",
    "KNOWLEDGE_DATABASE_URL",
    "QA_DATABASE_URL",
    "DOCUMENT_DATABASE_URL",
    "AI_GATEWAY_DATABASE_URL",
    "001-local-demo-seed.sql",
    "002-ai-gateway-model-profiles.sql",
    "003-qa-document-mcp.sql",
    "004-qa-default-knowledge-base.sql",
    "render-ai-gateway-local-seed",
    "--runtime api",
    "--runtime full",
    "knowledge-runtime-api",
    "knowledge-runtime-worker",
    "startup status",
    "host process groups",
    "compose_cmd ps",
    "write_compose_env_value",
    "align_host_run_ai_gateway_models",
    "AI_GATEWAY_LOCAL_CHAT_MODEL for host-run QA",
    "AI_GATEWAY_LOCAL_CHAT_MODEL for host-run Document",
    "go mod download",
    "go run module@version",
]

FORBIDDEN_START_TOKENS = [
    "\nuv sync",
    "exec uv sync",
    "uv sync --",
    "download_deps.py",
    "go mod download)",
    "go install github.com/pressly/goose",
]

REQUIRED_STOP_TOKENS = [
    "[stop]",
    "[ok]",
    "[warn]",
    "[fail]",
    "[hint]",
    "completed successfully",
    "failed during",
    "nothing to stop",
    'kill -0 -- "-$pid"',
    'kill -TERM -- "-$pid"',
    'kill -KILL -- "-$pid"',
]

REQUIRED_CLEAN_TOKENS = [
    "[clean]",
    "./scripts/local/stop.sh",
    "down -v",
    "--remove-orphans",
    ".local/tools/config-ctl",
    "Images, source files, .env.local, .local/tools, and .local/bin are not removed.",
]

REQUIRED_ENV_TOKENS = [
    "POSTGRES_PASSWORD=local-demo-postgres-password",
    "MINIO_ROOT_USER=minio_local_demo",
    "MINIO_ROOT_PASSWORD=minio-local-demo-password",
    "INTERNAL_SERVICE_TOKEN=local-dev-internal-service-token-change-me",
    "AUTH_GATEWAY_ADMIN_SERVICE_TOKEN=local-dev-gateway-admin-token-change-me",
    "GATEWAY_AUTH_ADMIN_SERVICE_TOKEN=local-dev-gateway-admin-token-change-me",
    "TOKEN_HASH_SECRET=local-demo-token-hash-secret-change-me",
    "AI_GATEWAY_SERVICE_TOKEN_HASHES=sha256:",
    "AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY=local-demo-credential-key-change-me",
    "AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY_REF=local-demo-key-v1",
    "LOCAL_ADMIN_USERNAME=admin",
    "LOCAL_ADMIN_PASSWORD=LocalDemoAdmin#12345",
    "LOCAL_SUPER_ADMIN_USERNAME=superadmin",
    "LOCAL_SUPER_ADMIN_PASSWORD=LocalDemoAdmin#12345",
    "POSTGRES_ADMIN_URL=postgres://postgres:local-demo-postgres-password@localhost:5432/postgres?sslmode=disable",
    "AUTH_DATABASE_URL=postgres://auth_app:auth_app_dev@localhost:5432/auth_system?sslmode=disable",
    "FILE_DATABASE_URL=postgres://file_app:file_app_dev@localhost:5432/file_system?sslmode=disable",
    "KNOWLEDGE_DATABASE_URL=postgres://knowledge_app:knowledge_app_dev@localhost:5432/knowledge_system?sslmode=disable",
    "QA_DATABASE_URL=postgres://qa_app:qa_app_dev@localhost:5432/qa_system?sslmode=disable",
    "DOCUMENT_DATABASE_URL=postgres://document_app:document_app_dev@localhost:5432/document_system?sslmode=disable",
    "AI_GATEWAY_DATABASE_URL=postgres://ai_gateway_app:ai_gateway_app_dev@localhost:5432/ai_gateway_system?sslmode=disable",
    "FILE_MINIO_ACCESS_KEY=minio_local_demo",
    "FILE_MINIO_SECRET_KEY=minio-local-demo-password",
    "KNOWLEDGE_SERVICE_TOKEN=local-dev-internal-service-token-change-me",
    "AI_GATEWAY_SERVICE_TOKEN=local-dev-internal-service-token-change-me",
    "AI_GATEWAY_TOKEN=local-dev-internal-service-token-change-me",
    "MCP_SERVER_TOKEN=local-dev-internal-service-token-change-me",
    "DOCUMENT_FILE_SERVICE_TOKEN=local-dev-internal-service-token-change-me",
    "DOCUMENT_AI_GATEWAY_SERVICE_TOKEN=local-dev-internal-service-token-change-me",
    "DOCUMENT_KNOWLEDGE_SERVICE_TOKEN=local-dev-internal-service-token-change-me",
    "VENDOR_RUNTIME_SERVICE_TOKEN=local-dev-runtime-service-token-change-me",
    "KNOWLEDGE_RUNTIME_SERVICE_TOKEN=local-dev-runtime-service-token-change-me",
    "KNOWLEDGE_RUNTIME_AI_GATEWAY_SERVICE_TOKEN=local-dev-internal-service-token-change-me",
    "AI_GATEWAY_LOCAL_SEED_ENABLED=false",
    "# AI_GATEWAY_LOCAL_PROVIDER=siliconflow",
    "# AI_GATEWAY_LOCAL_PROVIDER_BASE_URL=https://api.siliconflow.cn/v1",
    "# AI_GATEWAY_LOCAL_PROVIDER_API_KEY=<local-provider-api-key>",
    "# AI_GATEWAY_LOCAL_CHAT_MODEL=deepseek-ai/DeepSeek-V4-Flash",
    "# AI_GATEWAY_LOCAL_EMBEDDING_MODEL=BAAI/bge-m3",
    "# AI_GATEWAY_LOCAL_EMBEDDING_DIMENSIONS=1024",
    "# AI_GATEWAY_LOCAL_RERANK_MODEL=BAAI/bge-reranker-v2-m3",
    "# AI_GATEWAY_LOCAL_RERANK_TOP_N=5",
    "# UV_DEFAULT_INDEX=https://pypi.tuna.tsinghua.edu.cn/simple",
    "# GOPROXY=https://goproxy.cn,direct",
    "# GOSUMDB=sum.golang.google.cn",
]

REQUIRED_CONFIG_TOKENS = [
    "COMPOSE_PROJECT_NAME:",
    "POSTGRES_IMAGE:",
    "value: postgres:16-alpine",
    "REDIS_IMAGE:",
    "value: redis:7-alpine",
    "MINIO_IMAGE:",
    "value: minio/minio:RELEASE.2025-09-07T16-13-09Z",
    "MINIO_MC_IMAGE:",
    "value: minio/mc:RELEASE.2025-08-13T08-35-41Z",
    "UV_DEFAULT_INDEX:",
    "value: https://pypi.org/simple",
    "GOPROXY:",
    "value: https://proxy.golang.org,direct",
    "GOSUMDB:",
    "value: sum.golang.org",
    "DB_TYPE:",
    "value: postgres",
    "MCP_TRANSPORT:",
    "MCP_SERVER_ALIAS:",
    "MCP_SERVER_URL:",
    "MCP_SERVER_TOKEN_HEADER:",
    "VENDOR_RUNTIME_URL:",
    "value: http://127.0.0.1:9380",
    "KNOWLEDGE_RUNTIME_READINESS_MODE:",
    "KNOWLEDGE_AUTO_START_INGESTION:",
    "KNOWLEDGE_RUNTIME_WORKER_IDLE_SHUTDOWN_SECONDS:",
    "KNOWLEDGE_RUNTIME_WORKER_IDLE_CHECK_SECONDS:",
    "DOC_ENGINE:",
    "KNOWLEDGE_RUNTIME_ES_URL:",
    "value: http://127.0.0.1:9200",
    "KNOWLEDGE_RUNTIME_ELASTICSEARCH_IMAGE:",
    "value: docker.elastic.co/elasticsearch/elasticsearch:8.15.3",
    "MODEL_ID:",
    "DOCUMENT_AI_GATEWAY_MODEL:",
    "KNOWLEDGE_VENDOR_EMBEDDING_ID:",
    "value: BAAI/bge-m3@default@AI_GATEWAY",
    "KNOWLEDGE_VENDOR_RERANK_ID:",
    "value: BAAI/bge-reranker-v2-m3@default@AI_GATEWAY",
]

FORBIDDEN_STARTUP_DOC_TOKENS = [
    "export AUTH_DATABASE_URL",
    "export FILE_DATABASE_URL",
    "export KNOWLEDGE_DATABASE_URL",
    "export QA_DATABASE_URL",
    "export DOCUMENT_DATABASE_URL",
    "export AI_GATEWAY_DATABASE_URL",
    "docker compose up --build",
    "docker compose --profile ai",
    "seed-local",
]

FORBIDDEN_PATTERNS = [
    (re.compile(r"sk-[A-Za-z0-9_-]{16,}"), "OpenAI-style API key"),
    (re.compile(r"AKIA[0-9A-Z]{16}"), "AWS access key"),
    (re.compile(r"AIza[0-9A-Za-z_-]{20,}"), "Google API key"),
    (re.compile(r"-----BEGIN (?:RSA |EC |OPENSSH |)PRIVATE KEY-----"), "private key"),
    (re.compile(r"(?i)\bproduction\b.*\bpassword\b"), "production password wording"),
    (re.compile(r"(?i)\bminio(?:_|-)?secret(?:_|-)?key\b\s*[:=]\s*['\"]?[A-Za-z0-9+/]{12,}"), "MinIO secret key"),
]


def verify_local_seed_contract(root: Path) -> list[str]:
    root = root.resolve()
    issues: list[str] = []

    seed_001 = read_required(root, SEED_001, issues)
    seed_002 = read_required(root, SEED_002, issues)
    seed_003 = read_required(root, SEED_003, issues)
    seed_004 = read_required(root, SEED_004, issues)
    cleanup_seed = read_required(root, CLEANUP_SEED, issues)
    auth_migrations = read_required_glob(root, AUTH_MIGRATIONS_DIR, "*.sql", issues)
    deploy_readme = read_required(root, DEPLOY_README, issues)
    runbook = read_required(root, LOCAL_RUNBOOK, issues)
    env_example = read_required(root, ENV_EXAMPLE, issues)
    config_readme = read_required(root, CONFIG_README, issues)
    config_base = read_required(root, CONFIG_BASE, issues)
    check_script = read_required(root, CHECK_SCRIPT, issues)
    start_script = read_required(root, START_SCRIPT, issues)
    clean_script = read_required(root, CLEAN_SCRIPT, issues)
    ai_gateway_local_seed_renderer = read_required(root, AI_GATEWAY_LOCAL_SEED_RENDERER, issues)
    stop_script = read_required(root, STOP_SCRIPT, issues)
    local_common_helper = read_required(root, LOCAL_COMMON_HELPER, issues)
    local_process_helper = read_required(root, LOCAL_PROCESS_HELPER, issues)
    local_knowledge_runtime_helper = read_required(root, LOCAL_KNOWLEDGE_RUNTIME_HELPER, issues)
    ai_gateway_local_seed_main = read_required(root, AI_GATEWAY_LOCAL_SEED_MAIN, issues)
    gitignore = read_required(root, GITIGNORE, issues)

    issues.extend(validate_seed_001(seed_001))
    issues.extend(validate_seed_002(seed_002))
    issues.extend(validate_seed_003(seed_003))
    issues.extend(validate_seed_004(seed_004))
    issues.extend(validate_cleanup_seed(cleanup_seed))
    issues.extend(validate_auth_migrations(auth_migrations))
    issues.extend(
        validate_docs(
            deploy_readme,
            runbook,
            env_example,
            config_readme,
            config_base,
            check_script,
            start_script,
            clean_script,
            ai_gateway_local_seed_renderer,
            stop_script,
            ai_gateway_local_seed_main,
            local_common_helper,
            local_process_helper,
            local_knowledge_runtime_helper,
        )
    )
    issues.extend(validate_gitignore(gitignore))
    issues.extend(validate_executable_entrypoints(root))
    issues.extend(validate_forbidden_content(root))
    return issues


def read_required(root: Path, relative: Path, issues: list[str]) -> str:
    path = root / relative
    try:
        return path.read_text(encoding="utf-8")
    except OSError as exc:
        issues.append(f"{relative} is required but cannot be read: {exc}")
        return ""


def read_required_glob(root: Path, relative: Path, pattern: str, issues: list[str]) -> str:
    path = root / relative
    try:
        files = sorted(path.glob(pattern))
    except OSError as exc:
        issues.append(f"{relative} is required but cannot be read: {exc}")
        return ""
    if not files:
        issues.append(f"{relative} must contain `{pattern}` files")
        return ""

    contents: list[str] = []
    for file in files:
        try:
            contents.append(file.read_text(encoding="utf-8"))
        except OSError as exc:
            issues.append(f"{file.relative_to(root)} is required but cannot be read: {exc}")
    return "\n".join(contents)


def validate_seed_001(content: str) -> list[str]:
    if not content:
        return []
    issues: list[str] = []
    for pattern in REQUIRED_DATABASE_SECTIONS:
        if not re.search(pattern, content):
            issues.append(f"{SEED_001} missing database section matching `{pattern}`")
    for group, tokens in REQUIRED_SEED_001_TOKENS.items():
        for token in tokens:
            if token not in content:
                issues.append(f"{SEED_001} missing {group} token `{token}`")
    if content.count("ON CONFLICT") < 10:
        issues.append(f"{SEED_001} should use ON CONFLICT for deterministic idempotent rows")
    if "file_ref" in content.lower() and "file_ref,\n    filename" not in content:
        issues.append(f"{SEED_001} should keep demo file_ref fields explicitly null")
    return issues


def validate_seed_002(content: str) -> list[str]:
    if not content:
        return []
    issues: list[str] = []
    for token in REQUIRED_AI_TOKENS:
        if token not in content:
            issues.append(f"{SEED_002} missing AI placeholder token `{token}`")
    for token in FORBIDDEN_AI_TOKENS:
        if token in content:
            issues.append(f"{SEED_002} must not use container-only host token `{token}`")
    if content.count("ON CONFLICT") < 2:
        issues.append(f"{SEED_002} should use ON CONFLICT for model profiles and credentials")
    return issues


def validate_seed_003(content: str) -> list[str]:
    if not content:
        return []
    issues: list[str] = []
    for token in REQUIRED_DOCUMENT_MCP_TOKENS:
        if token.startswith(r"\\connect"):
            if not re.search(token, content):
                issues.append(f"{SEED_003} missing database section matching `{token}`")
        elif token not in content:
            issues.append(f"{SEED_003} missing Document MCP token `{token}`")
    if "token_encrypted" not in content or "NULL" not in content:
        issues.append(f"{SEED_003} must keep the Document MCP credential out of PostgreSQL")
    return issues


def validate_seed_004(content: str) -> list[str]:
    if not content:
        return []
    issues: list[str] = []
    for token in REQUIRED_QA_DEFAULT_KB_TOKENS:
        if token.startswith(r"\\connect"):
            if not re.search(token, content):
                issues.append(f"{SEED_004} missing database section matching `{token}`")
        elif token not in content:
            issues.append(f"{SEED_004} missing QA global-search seed token `{token}`")
    return issues


def validate_cleanup_seed(content: str) -> list[str]:
    if not content:
        return []
    issues: list[str] = []
    for token in [
        "usr_local_admin",
        "usr_local_super_admin",
        "doc_local_demo_seed",
        "22222222-2222-4222-8222-222222222301",
        "33333333-3333-4333-8333-333333333301",
        "33333333-3333-4333-8333-333333333601",
        "kb_local_demo",
    ]:
        if token not in content:
            issues.append(f"{CLEANUP_SEED} missing cleanup token `{token}`")
    for table in [
        "mcp_servers",
        "qa_config_knowledge_bases",
        "message_content_blocks",
        "report_section_versions",
        "document_chunks",
        "auth_credentials",
    ]:
        if table not in content:
            issues.append(f"{CLEANUP_SEED} missing cleanup table `{table}`")
    return issues


def validate_auth_migrations(content: str) -> list[str]:
    if not content:
        return []
    issues: list[str] = []
    for group, tokens in REQUIRED_AUTH_MIGRATION_TOKENS.items():
        for token in tokens:
            if token not in content:
                issues.append(f"{AUTH_MIGRATIONS_DIR} missing {group} token `{token}`")
    return issues


def validate_docs(
    deploy_readme: str,
    runbook: str,
    env_example: str,
    config_readme: str,
    config_base: str,
    check_script: str,
    start_script: str,
    clean_script: str,
    ai_gateway_local_seed_renderer: str,
    stop_script: str,
    ai_gateway_local_seed_main: str,
    local_common_helper: str = "",
    local_process_helper: str = "",
    local_knowledge_runtime_helper: str = "",
) -> list[str]:
    issues: list[str] = []
    combined = "\n".join([deploy_readme, runbook, env_example, config_readme])
    start_contract = "\n".join([start_script, local_common_helper, local_process_helper, local_knowledge_runtime_helper])
    stop_contract = "\n".join([stop_script, local_process_helper])
    for token in REQUIRED_DOC_TOKENS:
        if token not in combined:
            issues.append(f"seed documentation missing `{token}`")
    for token in REQUIRED_ENV_TOKENS:
        if token not in env_example:
            issues.append(f"{ENV_EXAMPLE} missing local default `{token}`")
    for token in REQUIRED_CONFIG_TOKENS:
        if token not in config_base:
            issues.append(f"{CONFIG_BASE} missing committed config default `{token}`")
    for token in FORBIDDEN_STARTUP_DOC_TOKENS:
        if token in combined:
            issues.append(f"startup documentation must not include `{token}`")
    for token in REQUIRED_CHECK_TOKENS:
        if token not in check_script:
            issues.append(f"{CHECK_SCRIPT} missing check token `{token}`")
    for token in FORBIDDEN_CHECK_TOKENS:
        if token in check_script:
            issues.append(f"{CHECK_SCRIPT} must not contain preflight Docker state token `{token}`")
    for token in REQUIRED_START_TOKENS:
        if token not in start_contract:
            issues.append(f"{START_SCRIPT} missing startup token `{token}`")
    for token in FORBIDDEN_START_TOKENS:
        if token in start_script:
            issues.append(f"{START_SCRIPT} must not contain startup download token `{token}`")
    for token in REQUIRED_CLEAN_TOKENS:
        if token not in clean_script:
            issues.append(f"{CLEAN_SCRIPT} missing cleanup token `{token}`")
    for token in [
        "AI_GATEWAY_LOCAL_PROVIDER",
        "AI_GATEWAY_LOCAL_PROVIDER_BASE_URL",
        "AI_GATEWAY_LOCAL_PROVIDER_API_KEY",
        "AI_GATEWAY_LOCAL_CHAT_MODEL",
        "AI_GATEWAY_LOCAL_EMBEDDING_MODEL",
        "AI_GATEWAY_LOCAL_EMBEDDING_DIMENSIONS",
        "AI_GATEWAY_LOCAL_RERANK_MODEL",
        "AI_GATEWAY_LOCAL_RERANK_TOP_N",
        "provider_credentials",
        "llm_config_versions",
        "fingerprintContext",
    ]:
        if token not in ai_gateway_local_seed_renderer:
            issues.append(f"{AI_GATEWAY_LOCAL_SEED_RENDERER} missing local overlay token `{token}`")
    for token in REQUIRED_STOP_TOKENS:
        if token not in stop_contract:
            issues.append(f"{STOP_SCRIPT} missing stop token `{token}`")
    for token in [
        "QA_DATABASE_URL",
        "llm_config_versions",
        "syncQALLMConfig",
        "provider = 'ai-gateway'",
        "default-chat",
        "AI_GATEWAY_LOCAL_CHAT_MODEL",
    ]:
        if token not in ai_gateway_local_seed_main:
            issues.append(f"{AI_GATEWAY_LOCAL_SEED_MAIN} missing QA LLM sync token `{token}`")
    return issues


def validate_gitignore(content: str) -> list[str]:
    if not content:
        return []
    issues: list[str] = []
    for token in ["/.local/", "DL_T_673-1999.pdf"]:
        if token not in content:
            issues.append(f"{GITIGNORE} missing local runtime ignore token `{token}`")
    return issues


def validate_executable_entrypoints(root: Path) -> list[str]:
    issues: list[str] = []
    for relative in PUBLIC_LOCAL_ENTRYPOINTS:
        path = root / relative
        try:
            mode = path.stat().st_mode
        except OSError:
            continue
        if mode & 0o111 == 0:
            issues.append(f"{relative} must be executable because docs run it as ./{relative}")
    return issues


def validate_forbidden_content(root: Path) -> list[str]:
    issues: list[str] = []
    for relative in [
        SEED_001,
        SEED_002,
        SEED_003,
        SEED_004,
        CLEANUP_SEED,
        DEPLOY_README,
        LOCAL_RUNBOOK,
        ENV_EXAMPLE,
        CHECK_SCRIPT,
        START_SCRIPT,
        CLEAN_SCRIPT,
        AI_GATEWAY_LOCAL_SEED_RENDERER,
        LOCAL_COMMON_HELPER,
        LOCAL_PROCESS_HELPER,
        LOCAL_KNOWLEDGE_RUNTIME_HELPER,
        AI_GATEWAY_LOCAL_SEED_MAIN,
        STOP_SCRIPT,
    ]:
        path = root / relative
        if not path.exists():
            continue
        content = path.read_text(encoding="utf-8", errors="replace")
        for pattern, label in FORBIDDEN_PATTERNS:
            if pattern.search(content):
                issues.append(f"{relative} contains forbidden {label} pattern")
    return issues


def build_arg_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "root",
        nargs="?",
        type=Path,
        default=Path("."),
        help="Repository root to verify.",
    )
    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_arg_parser().parse_args(argv)
    issues = verify_local_seed_contract(args.root)
    if issues:
        print("Local seed contract verification failed:")
        for issue in issues:
            print(f"- {issue}")
        return 1
    print("Local seed contract verification passed.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
