import os
import stat
import subprocess
import tempfile
import textwrap
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]


class CloudDockerStartScriptTests(unittest.TestCase):
    def test_cloud_compose_does_not_use_seed_model_as_runtime_fallback(self) -> None:
        compose = (REPO_ROOT / "deploy" / "docker-compose.cloud.yml").read_text(encoding="utf-8")
        env_example = (REPO_ROOT / "deploy" / "docker" / "cloud.env.example").read_text(encoding="utf-8")

        self.assertIn("DOCUMENT_AI_GATEWAY_MODEL: ${DOCUMENT_AI_GATEWAY_MODEL:-}", compose)
        self.assertIn("MODEL_ID: ${MODEL_ID:-}", compose)
        self.assertNotIn("DOCUMENT_AI_GATEWAY_MODEL:-${AI_GATEWAY_LOCAL_CHAT_MODEL", compose)
        self.assertNotIn("MODEL_ID:-${AI_GATEWAY_LOCAL_CHAT_MODEL", compose)
        self.assertIn("AI_GATEWAY_LOCAL_CHAT_MODEL=", env_example)
        self.assertNotIn("AI_GATEWAY_LOCAL_CHAT_MODEL=<cloud-chat-model>", env_example)

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
        result, docker_calls = self.run_start(
            self.base_cloud_env()
            + """
            DOCKER_SEED_ENABLED=true
            """
        )

        self.assertNotEqual(0, result.returncode)
        self.assertIn("POSTGRES_ADMIN_URL", result.stderr)
        self.assertIn("PADDLEOCR_ACCESS_TOKEN", result.stderr)
        self.assertIn("AI_GATEWAY_LOCAL_PROVIDER_BASE_URL", result.stderr)
        self.assertIn("AI_GATEWAY_LOCAL_PROVIDER_API_KEY", result.stderr)
        self.assertIn("AI_GATEWAY_LOCAL_CHAT_MODEL", result.stderr)
        self.assertIn("info", docker_calls)
        self.assertNotIn("compose", docker_calls)

    def test_start_rejects_local_demo_placeholders_when_seed_is_enabled(self) -> None:
        result, docker_calls = self.run_start(
            self.base_cloud_env()
            + """
            DOCKER_SEED_ENABLED=true
            INTERNAL_SERVICE_TOKEN=local-dev-internal-service-token-change-me
            TOKEN_HASH_SECRET=local-demo-token-hash-secret-change-me
            AI_GATEWAY_SERVICE_TOKEN_HASHES=sha256:26c6719c056dabe8530ea09f1e8f7593cbcf98a060731c0fc786a5eb48e71ce7
            POSTGRES_ADMIN_URL=postgres://postgres:secret@postgres.example:5432/postgres?sslmode=require
            PADDLEOCR_ACCESS_TOKEN=cloud-ocr-token
            AI_GATEWAY_LOCAL_PROVIDER_BASE_URL=https://provider.example/v1
            AI_GATEWAY_LOCAL_PROVIDER_API_KEY=cloud-provider-key
            AI_GATEWAY_LOCAL_CHAT_MODEL=cloud-chat
            """
        )

        self.assertNotEqual(0, result.returncode)
        self.assertIn("replace local/demo placeholder values", result.stderr)
        self.assertIn("INTERNAL_SERVICE_TOKEN", result.stderr)
        self.assertIn("TOKEN_HASH_SECRET", result.stderr)
        self.assertIn("AI_GATEWAY_SERVICE_TOKEN_HASHES", result.stderr)
        self.assertIn("info", docker_calls)
        self.assertNotIn("compose", docker_calls)

    def test_start_does_not_require_provider_seed_values_when_provider_seed_is_disabled(self) -> None:
        result, _ = self.run_start(
            self.base_cloud_env()
            + """
            DOCKER_SEED_ENABLED=true
            POSTGRES_ADMIN_URL=postgres://postgres:secret@postgres.example:5432/postgres?sslmode=require
            PADDLEOCR_ACCESS_TOKEN=ocr-token
            AI_GATEWAY_LOCAL_SEED_ENABLED=false
            """
        )

        self.assertEqual(0, result.returncode, result.stderr)

    def test_seed_entrypoint_skips_by_default(self) -> None:
        result = subprocess.run(
            ["sh", str(REPO_ROOT / "deploy" / "docker" / "full" / "seed.sh")],
            cwd=REPO_ROOT,
            env={},
            text=True,
            capture_output=True,
            check=False,
        )

        self.assertEqual(0, result.returncode, result.stderr)
        self.assertIn("docker seed: skipped", result.stdout)

    def test_seed_entrypoint_rejects_local_demo_placeholders(self) -> None:
        result = subprocess.run(
            ["sh", str(REPO_ROOT / "deploy" / "docker" / "full" / "seed.sh")],
            cwd=REPO_ROOT,
            env={
                "DOCKER_SEED_ENABLED": "true",
                "INTERNAL_SERVICE_TOKEN": "local-dev-internal-service-token-change-me",
                "AUTH_GATEWAY_ADMIN_SERVICE_TOKEN": "cloud-auth-admin-service-token",
                "GATEWAY_AUTH_ADMIN_SERVICE_TOKEN": "cloud-auth-admin-service-token",
                "TOKEN_HASH_SECRET": "cloud-token-hash-secret",
                "AI_GATEWAY_SERVICE_TOKEN_HASHES": "sha256:674d4e99fda4188e1180e9fc9cf5c98efd4d19ecb84b4a840f1b6f80e0bb92ca",
                "AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY": "cloud-ai-gateway-credential-encryption-key",
                "AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY_REF": "cloud-ai-gateway-key-v1",
                "POSTGRES_ADMIN_URL": "postgres://postgres:secret@postgres.example:5432/postgres?sslmode=require",
                "PADDLEOCR_ACCESS_TOKEN": "cloud-ocr-token",
                "AI_GATEWAY_LOCAL_PROVIDER_BASE_URL": "https://provider.example/v1",
                "AI_GATEWAY_LOCAL_PROVIDER_API_KEY": "cloud-provider-key",
                "AI_GATEWAY_LOCAL_CHAT_MODEL": "cloud-chat",
            },
            text=True,
            capture_output=True,
            check=False,
        )

        self.assertNotEqual(0, result.returncode)
        self.assertIn("INTERNAL_SERVICE_TOKEN must be replaced", result.stderr)

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
        INTERNAL_SERVICE_TOKEN=cloud-internal-service-token
        AUTH_GATEWAY_ADMIN_SERVICE_TOKEN=cloud-auth-admin-service-token
        GATEWAY_AUTH_ADMIN_SERVICE_TOKEN=cloud-auth-admin-service-token
        TOKEN_HASH_SECRET=cloud-token-hash-secret
        AI_GATEWAY_SERVICE_TOKEN_HASHES=sha256:674d4e99fda4188e1180e9fc9cf5c98efd4d19ecb84b4a840f1b6f80e0bb92ca
        AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY=cloud-ai-gateway-credential-encryption-key
        AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY_REF=cloud-ai-gateway-key-v1
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
        MCP_SERVER_TOKEN=cloud-document-mcp-token
        KNOWLEDGE_MCP_TOKEN=cloud-knowledge-mcp-token
        """


if __name__ == "__main__":
    unittest.main()
