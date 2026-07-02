#!/usr/bin/env python3
"""Verify repository Docker and Compose files follow local pull-only policy."""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path


EXPECTED_ENV_IMAGE_OVERRIDES = {
    "POSTGRES_IMAGE": "docker.m.daocloud.io/library/postgres:16-alpine",
    "REDIS_IMAGE": "docker.m.daocloud.io/library/redis:7-alpine",
    "QDRANT_IMAGE": "docker.m.daocloud.io/qdrant/qdrant:v1.18.2",
    "MINIO_IMAGE": "docker.m.daocloud.io/minio/minio:RELEASE.2025-09-07T16-13-09Z",
    "MINIO_MC_IMAGE": "docker.m.daocloud.io/minio/mc:RELEASE.2025-08-13T08-35-41Z",
}
EXPECTED_IMAGE_DEFAULTS = {
    "POSTGRES_IMAGE": "postgres:16-alpine",
    "REDIS_IMAGE": "redis:7-alpine",
    "QDRANT_IMAGE": "qdrant/qdrant:v1.18.2",
    "MINIO_IMAGE": "minio/minio:RELEASE.2025-09-07T16-13-09Z",
    "MINIO_MC_IMAGE": "minio/mc:RELEASE.2025-08-13T08-35-41Z",
}
LOCAL_COMPOSE_FILE = Path("deploy/docker-compose.yml")
ALLOWED_LOCAL_COMPOSE_SERVICES = ("postgres", "redis", "qdrant", "minio", "minio-init")
DISALLOWED_LOCAL_COMPOSE_SERVICES = (
    "migrate-auth",
    "migrate-file",
    "migrate-knowledge",
    "migrate-qa",
    "migrate-document",
    "migrate-ai-gateway",
    "seed-local",
    "seed-local-ai",
    "auth",
    "file",
    "parser",
    "knowledge",
    "qa",
    "document",
    "ai-gateway",
    "gateway",
)


def verify_docker_policy(root: Path) -> list[str]:
    issues: list[str] = []
    issues.extend(validate_compose_file(root, root / LOCAL_COMPOSE_FILE))
    issues.extend(validate_env_example(root))
    return issues


def validate_compose_file(root: Path, compose_file: Path) -> list[str]:
    rel = compose_file.as_posix()
    if compose_file.is_absolute():
        rel = compose_file.relative_to(root).as_posix()
    content = compose_file.read_text(encoding="utf-8")
    issues: list[str] = []

    for line_no, image in collect_compose_images(content):
        if uses_latest_tag(image):
            issues.append(f"{rel}:{line_no}: Compose image `{image}` must not use latest")
        issues.extend(validate_compose_image_default(rel, line_no, image))

    if rel == "deploy/docker-compose.yml":
        issues.extend(validate_local_infra_compose(rel, content))

    return issues


def collect_compose_images(content: str) -> list[tuple[int, str]]:
    images: list[tuple[int, str]] = []
    for line_no, line in enumerate(content.splitlines(), start=1):
        match = re.match(r"^\s*image:\s*(.+?)\s*(?:#.*)?$", line)
        if not match:
            continue
        image = match.group(1).strip().strip("'\"")
        images.append((line_no, image))
    return images


def validate_compose_image_default(rel: str, line_no: int, image: str) -> list[str]:
    issues: list[str] = []
    for variable, expected_default in EXPECTED_IMAGE_DEFAULTS.items():
        expected_expr = f"${{{variable}:-{expected_default}}}"
        if variable in image:
            if image != expected_expr:
                issues.append(
                    f"{rel}:{line_no}: `{variable}` must default to pinned `{expected_default}`"
                )
            return issues
        if image == expected_default:
            issues.append(
                f"{rel}:{line_no}: `{expected_default}` must be exposed through `{expected_expr}`"
            )
            return issues
    if re.match(r"^[^$].*:[^@]+$", image) and not image.startswith("${"):
        issues.append(
            f"{rel}:{line_no}: Compose image `{image}` must be exposed through a pinned override variable"
        )
    return issues


def validate_local_infra_compose(rel: str, content: str) -> list[str]:
    issues: list[str] = []
    services = tuple(extract_compose_services(content))
    service_set = set(services)
    if service_set != set(ALLOWED_LOCAL_COMPOSE_SERVICES):
        issues.append(
            f"{rel}: local Docker must only define infrastructure services "
            f"{', '.join(ALLOWED_LOCAL_COMPOSE_SERVICES)}; found {', '.join(services) or '(none)'}"
        )
    if re.search(r"(?m)^\s+build\s*:", content):
        issues.append(f"{rel}: local Docker is pull-only and must not contain build entries")
    for service in DISALLOWED_LOCAL_COMPOSE_SERVICES:
        if service in service_set:
            issues.append(f"{rel}: business service `{service}` must run on the host, not in local Docker")
    return issues


def extract_compose_services(content: str) -> list[str]:
    services: list[str] = []
    in_services = False
    for line in content.splitlines():
        if re.match(r"^services:\s*$", line):
            in_services = True
            continue
        if in_services and re.match(r"^[A-Za-z0-9_.-]+:\s*$", line):
            break
        if not in_services:
            continue
        match = re.match(r"^  ([A-Za-z0-9_.-]+):\s*$", line)
        if match:
            services.append(match.group(1))
    return services


def validate_env_example(root: Path) -> list[str]:
    env_file = root / "deploy" / ".env.example"
    if not env_file.exists():
        return ["deploy/.env.example is required for local startup defaults"]

    content = env_file.read_text(encoding="utf-8")
    values = parse_env_file(content)
    issues: list[str] = []

    for key, expected in EXPECTED_ENV_IMAGE_OVERRIDES.items():
        actual = values.get(key)
        if actual != expected:
            issues.append(
                f"deploy/.env.example: `{key}` must stay `{expected}` for the documented mainland China Docker path"
            )

    for key, value in values.items():
        if key.endswith("_IMAGE") and uses_latest_tag(value):
            issues.append(f"deploy/.env.example: `{key}` must not use latest")

    return issues


def parse_env_file(content: str) -> dict[str, str]:
    values: dict[str, str] = {}
    for line in content.splitlines():
        stripped = line.strip()
        if not stripped or stripped.startswith("#") or "=" not in stripped:
            continue
        key, value = stripped.split("=", 1)
        values[key.strip()] = value.strip().strip("'\"")
    return values


def uses_latest_tag(image: str) -> bool:
    return ":latest" in image or ":-latest" in image or image.endswith(":latest")


def has_explicit_tag_or_digest(image: str) -> bool:
    if "@" in image:
        return True
    last_component = image.rsplit("/", maxsplit=1)[-1]
    return ":" in last_component


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--root",
        type=Path,
        default=Path.cwd(),
        help="repository root; defaults to current working directory",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    root = args.root.resolve()
    issues = verify_docker_policy(root)
    if issues:
        for issue in issues:
            print(f"- {issue}", file=sys.stderr)
        return 1
    print("Docker policy checks passed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
