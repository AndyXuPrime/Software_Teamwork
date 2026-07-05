import os
import stat
import subprocess
import tempfile
import textwrap
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]


class CloudDockerStartScriptTests(unittest.TestCase):
    def test_start_allows_preseeded_cloud_stack_without_seed_only_values(self) -> None:
        result, docker_calls = self.run_start(
            self.base_cloud_env()
            + """
            DOCKER_SEED_ENABLED=false
            """
        )

        self.assertEqual(0, result.returncode, result.stderr)
        self.assertIn("info", docker_calls)
        self.assertIn("compose -f", docker_calls)
        self.assertIn("config --quiet", docker_calls)
        self.assertIn("up -d --build", docker_calls)

    def test_start_requires_seed_values_when_seed_is_enabled(self) -> None:
        result, docker_calls = self.run_start(self.base_cloud_env())

        self.assertNotEqual(0, result.returncode)
        self.assertIn("POSTGRES_ADMIN_URL", result.stderr)
        self.assertIn("PADDLEOCR_ACCESS_TOKEN", result.stderr)
        self.assertIn("AI_GATEWAY_LOCAL_PROVIDER_BASE_URL", result.stderr)
        self.assertIn("AI_GATEWAY_LOCAL_PROVIDER_API_KEY", result.stderr)
        self.assertIn("AI_GATEWAY_LOCAL_CHAT_MODEL", result.stderr)
        self.assertIn("info", docker_calls)
        self.assertNotIn("compose", docker_calls)

    def test_start_does_not_require_provider_seed_values_when_provider_seed_is_disabled(self) -> None:
        result, _ = self.run_start(
            self.base_cloud_env()
            + """
            POSTGRES_ADMIN_URL=postgres://postgres:secret@postgres.example:5432/postgres?sslmode=require
            PADDLEOCR_ACCESS_TOKEN=ocr-token
            AI_GATEWAY_LOCAL_SEED_ENABLED=false
            """
        )

        self.assertEqual(0, result.returncode, result.stderr)

    def run_start(self, env_content: str) -> tuple[subprocess.CompletedProcess[str], str]:
        with tempfile.TemporaryDirectory() as directory:
            tmp = Path(directory)
            env_file = tmp / "cloud.env"
            env_file.write_text(textwrap.dedent(env_content).strip() + "\n", encoding="utf-8")
            fake_bin = tmp / "bin"
            fake_bin.mkdir()
            docker_log = tmp / "docker.log"
            docker = fake_bin / "docker"
            docker.write_text(
                textwrap.dedent(
                    """\
                    #!/usr/bin/env bash
                    set -euo pipefail
                    printf '%s\\n' "$*" >> "$FAKE_DOCKER_LOG"
                    case "${1:-}" in
                      info) exit 0 ;;
                      compose) exit 0 ;;
                    esac
                    printf 'unexpected docker command: %s\\n' "$*" >&2
                    exit 2
                    """
                ),
                encoding="utf-8",
            )
            docker.chmod(docker.stat().st_mode | stat.S_IXUSR)

            result = subprocess.run(
                [str(REPO_ROOT / "scripts" / "docker" / "start.sh"), "--env-file", str(env_file)],
                cwd=REPO_ROOT,
                env={**os.environ, "PATH": f"{fake_bin}:{os.environ['PATH']}", "FAKE_DOCKER_LOG": str(docker_log)},
                text=True,
                capture_output=True,
                check=False,
            )
            docker_calls = docker_log.read_text(encoding="utf-8") if docker_log.exists() else ""
            return result, docker_calls

    def base_cloud_env(self) -> str:
        return """
        AUTH_DATABASE_URL=postgres://auth:secret@postgres.example:5432/auth_system?sslmode=require
        FILE_DATABASE_URL=postgres://file:secret@postgres.example:5432/file_system?sslmode=require
        KNOWLEDGE_DATABASE_URL=postgres://knowledge:secret@postgres.example:5432/knowledge_system?sslmode=require
        QA_DATABASE_URL=postgres://qa:secret@postgres.example:5432/qa_system?sslmode=require
        DOCUMENT_DATABASE_URL=postgres://document:secret@postgres.example:5432/document_system?sslmode=require
        AI_GATEWAY_DATABASE_URL=postgres://ai_gateway:secret@postgres.example:5432/ai_gateway_system?sslmode=require
        GATEWAY_REDIS_ADDR=redis.example:6379
        DOCUMENT_REDIS_ADDR=redis.example:6379
        FILE_MINIO_ENDPOINT=object.example:9000
        FILE_MINIO_ACCESS_KEY=access-key
        FILE_MINIO_SECRET_KEY=secret-key
        FILE_MINIO_BUCKET=software-teamwork-cloud
        VENDOR_RUNTIME_URL=https://runtime.example
        VENDOR_RUNTIME_SERVICE_TOKEN=runtime-token
        """


if __name__ == "__main__":
    unittest.main()
