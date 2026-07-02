import tempfile
import textwrap
import unittest
from pathlib import Path

from scripts.check_docker_policy import verify_docker_policy


VALID_COMPOSE = textwrap.dedent(
    """
    services:
      postgres:
        image: ${POSTGRES_IMAGE:-postgres:16-alpine}
      redis:
        image: ${REDIS_IMAGE:-redis:7-alpine}
      qdrant:
        image: ${QDRANT_IMAGE:-qdrant/qdrant:v1.18.2}
      minio:
        image: ${MINIO_IMAGE:-minio/minio:RELEASE.2025-09-07T16-13-09Z}
      minio-init:
        image: ${MINIO_MC_IMAGE:-minio/mc:RELEASE.2025-08-13T08-35-41Z}
    """
)


VALID_ENV = textwrap.dedent(
    """
    POSTGRES_IMAGE=docker.m.daocloud.io/library/postgres:16-alpine
    REDIS_IMAGE=docker.m.daocloud.io/library/redis:7-alpine
    QDRANT_IMAGE=docker.m.daocloud.io/qdrant/qdrant:v1.18.2
    MINIO_IMAGE=docker.m.daocloud.io/minio/minio:RELEASE.2025-09-07T16-13-09Z
    MINIO_MC_IMAGE=docker.m.daocloud.io/minio/mc:RELEASE.2025-08-13T08-35-41Z
    """
)


class DockerPolicyTests(unittest.TestCase):
    def test_valid_policy_files_have_no_issues(self) -> None:
        issues = self.verify(
            files={
                "deploy/docker-compose.yml": VALID_COMPOSE,
                "deploy/.env.example": VALID_ENV,
            }
        )

        self.assertEqual([], issues)

    def test_root_compose_must_be_infra_only(self) -> None:
        compose = VALID_COMPOSE.replace(
            "  qdrant:\n",
            "  gateway:\n    image: ${GATEWAY_IMAGE:-registry.example.com/gateway:local}\n  qdrant:\n",
        )

        issues = self.verify(files={"deploy/docker-compose.yml": compose})

        self.assertIssueContains(issues, "local Docker must only define infrastructure services")
        self.assertIssueContains(issues, "business service `gateway` must run on the host")

    def test_root_compose_must_not_build(self) -> None:
        compose = VALID_COMPOSE.replace(
            "  minio-init:\n    image:",
            "  parser:\n    build:\n      context: ../services/parser\n  minio-init:\n    image:",
        )

        issues = self.verify(files={"deploy/docker-compose.yml": compose})

        self.assertIssueContains(issues, "must not contain build entries")
        self.assertIssueContains(issues, "business service `parser` must run on the host")

    def test_compose_image_defaults_must_stay_pinned_and_overridable(self) -> None:
        compose = VALID_COMPOSE.replace(
            "image: ${POSTGRES_IMAGE:-postgres:16-alpine}",
            "image: postgres:latest",
        )

        issues = self.verify(files={"deploy/docker-compose.yml": compose})

        self.assertIssueContains(issues, "must not use latest")
        self.assertIssueContains(issues, "must be exposed through")

    def test_env_example_regressions_are_reported(self) -> None:
        env = VALID_ENV.replace(
            "POSTGRES_IMAGE=docker.m.daocloud.io/library/postgres:16-alpine",
            "POSTGRES_IMAGE=postgres:latest",
        )

        issues = self.verify(files={"deploy/.env.example": env})

        self.assertIssueContains(issues, "POSTGRES_IMAGE")
        self.assertIssueContains(issues, "must not use latest")

    def test_business_docker_artifacts_are_reported(self) -> None:
        issues = self.verify(
            files={
                "services/auth/Dockerfile": "FROM golang:1.25\n",
                "services/parser/docker-compose.yml": "services: {}\n",
                "deploy/docker-compose.production.yml": "services: {}\n",
                "deploy/compose.preview.yml": "services: {}\n",
            }
        )

        self.assertIssueContains(issues, "services/auth/Dockerfile")
        self.assertIssueContains(issues, "business service Dockerfile")
        self.assertIssueContains(issues, "services/parser/docker-compose.yml")
        self.assertIssueContains(issues, "service-level Compose file")
        self.assertIssueContains(issues, "deploy/docker-compose.production.yml")
        self.assertIssueContains(issues, "deploy/compose.preview.yml")
        self.assertIssueContains(issues, "non-root deploy Compose file")

    def verify(self, *, files: dict[str, str]) -> list[str]:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            all_files = dict(files)
            all_files.setdefault("deploy/docker-compose.yml", VALID_COMPOSE)
            all_files.setdefault("deploy/.env.example", VALID_ENV)
            for relative, content in all_files.items():
                path = root / relative
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(content, encoding="utf-8")
            return verify_docker_policy(root)

    def assertIssueContains(self, issues: list[str], expected: str) -> None:
        self.assertTrue(
            any(expected in issue for issue in issues),
            f"Expected issue containing {expected!r}, got: {issues!r}",
        )


if __name__ == "__main__":
    unittest.main()
