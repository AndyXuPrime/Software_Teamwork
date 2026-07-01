#!/usr/bin/env python3
"""Verify the root local/demo seed contract for integration fixtures."""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path


SEED_001 = Path("deploy/seeds/001-local-demo-seed.sql")
SEED_002 = Path("deploy/seeds/002-ai-gateway-model-profiles.sql")
CLEANUP_SEED = Path("deploy/seeds/099-local-demo-cleanup.sql")
COMPOSE_FILE = Path("deploy/docker-compose.yml")
DEPLOY_README = Path("deploy/README.md")
LOCAL_RUNBOOK = Path("docs/runbooks/local-integration.md")
ENV_EXAMPLE = Path("deploy/.env.example")

REQUIRED_SEED_001_TOKENS = {
    "Auth local admin user": ["usr_local_admin", "cred_local_admin_password", "urole_local_admin_admin"],
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
    "cred-local-chat",
    "cred-local-embedding",
    "cred-local-rerank",
    "local-demo-key-v1",
]

REQUIRED_DOC_TOKENS = [
    "seed-local",
    "seed-local-ai",
    "LOCAL_ADMIN_USERNAME=admin",
    "LOCAL_ADMIN_PASSWORD=LocalDemoAdmin#12345",
    "usr_local_admin",
    "kb_local_demo",
    "doc_local_demo_seed",
    "22222222-2222-4222-8222-222222222301",
    "33333333-3333-4333-8333-333333333301",
    "POST /api/v1/sessions",
    "/api/v1/admin/parser-configs",
    "admin:model-profile:write",
    "admin:parser-config:write",
    "argon2id",
    "rotation",
    "cleanup",
    "CI-safe",
    "local/manual",
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
    cleanup_seed = read_required(root, CLEANUP_SEED, issues)
    compose = read_required(root, COMPOSE_FILE, issues)
    deploy_readme = read_required(root, DEPLOY_README, issues)
    runbook = read_required(root, LOCAL_RUNBOOK, issues)
    env_example = read_required(root, ENV_EXAMPLE, issues)

    issues.extend(validate_seed_001(seed_001))
    issues.extend(validate_seed_002(seed_002))
    issues.extend(validate_cleanup_seed(cleanup_seed))
    issues.extend(validate_compose(compose))
    issues.extend(validate_docs(deploy_readme, runbook, env_example))
    issues.extend(validate_forbidden_content(root))
    return issues


def read_required(root: Path, relative: Path, issues: list[str]) -> str:
    path = root / relative
    try:
        return path.read_text(encoding="utf-8")
    except OSError as exc:
        issues.append(f"{relative} is required but cannot be read: {exc}")
        return ""


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
    if content.count("ON CONFLICT") < 2:
        issues.append(f"{SEED_002} should use ON CONFLICT for model profiles and credentials")
    return issues


def validate_cleanup_seed(content: str) -> list[str]:
    if not content:
        return []
    issues: list[str] = []
    for token in [
        "usr_local_admin",
        "doc_local_demo_seed",
        "22222222-2222-4222-8222-222222222301",
        "33333333-3333-4333-8333-333333333301",
    ]:
        if token not in content:
            issues.append(f"{CLEANUP_SEED} missing cleanup token `{token}`")
    for table in ["message_content_blocks", "report_section_versions", "document_chunks", "auth_credentials"]:
        if table not in content:
            issues.append(f"{CLEANUP_SEED} missing cleanup table `{table}`")
    return issues


def validate_compose(content: str) -> list[str]:
    if not content:
        return []
    issues: list[str] = []
    seed_local_block = extract_service_block(content, "seed-local")
    if "migrate-qa" not in seed_local_block:
        issues.append(f"{COMPOSE_FILE} seed-local depends_on must include migrate-qa")
    if "001-local-demo-seed.sql" not in seed_local_block:
        issues.append(f"{COMPOSE_FILE} seed-local must run 001-local-demo-seed.sql")
    return issues


def validate_docs(deploy_readme: str, runbook: str, env_example: str) -> list[str]:
    issues: list[str] = []
    combined = "\n".join([deploy_readme, runbook, env_example])
    for token in REQUIRED_DOC_TOKENS:
        if token not in combined:
            issues.append(f"seed documentation missing `{token}`")
    if "system:admin" not in combined and "admin runtime config" not in combined:
        issues.append("seed documentation must mention system:admin or admin runtime config permissions")
    if "099-local-demo-cleanup.sql" not in combined and "down -v" not in combined:
        issues.append("seed documentation must mention targeted cleanup or full volume reset")
    return issues


def validate_forbidden_content(root: Path) -> list[str]:
    issues: list[str] = []
    for relative in [SEED_001, SEED_002, CLEANUP_SEED, DEPLOY_README, LOCAL_RUNBOOK, ENV_EXAMPLE]:
        path = root / relative
        if not path.exists():
            continue
        content = path.read_text(encoding="utf-8", errors="replace")
        for pattern, label in FORBIDDEN_PATTERNS:
            if pattern.search(content):
                issues.append(f"{relative} contains forbidden {label} pattern")
    return issues


def extract_service_block(compose: str, service_name: str) -> str:
    lines = compose.splitlines()
    start = None
    for index, line in enumerate(lines):
        if re.match(rf"^\s{{2}}{re.escape(service_name)}:\s*$", line):
            start = index
            break
    if start is None:
        return ""

    collected = [lines[start]]
    for line in lines[start + 1 :]:
        if re.match(r"^\s{2}[A-Za-z0-9_-]+:\s*$", line):
            break
        collected.append(line)
    return "\n".join(collected)


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
