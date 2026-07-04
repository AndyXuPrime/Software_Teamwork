import importlib
import sys
import tempfile
import textwrap
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))


def load_verifier():
    try:
        return importlib.import_module("scripts.verify_local_seed_contract")
    except ModuleNotFoundError as exc:
        if exc.name == "scripts.verify_local_seed_contract":
            raise AssertionError("scripts.verify_local_seed_contract module is missing") from exc
        raise


class LocalSeedContractTests(unittest.TestCase):
    def test_repository_seed_contract_has_no_issues(self) -> None:
        verifier = load_verifier()

        issues = verifier.verify_local_seed_contract(Path.cwd())

        self.assertEqual([], issues)

    def test_verifier_reports_non_executable_public_entrypoint(self) -> None:
        verifier = load_verifier()
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            script = root / "scripts" / "local" / "start.sh"
            script.parent.mkdir(parents=True)
            script.write_text("#!/usr/bin/env bash\n", encoding="utf-8")
            script.chmod(0o644)

            issues = verifier.validate_executable_entrypoints(root)

            script.chmod(0o755)
            fixed_issues = verifier.validate_executable_entrypoints(root)

        self.assertIssueContains(issues, "scripts/local/start.sh must be executable")
        self.assertEqual([], fixed_issues)

    def test_verifier_reports_missing_required_resource_ids(self) -> None:
        verifier = load_verifier()
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "deploy" / "seeds").mkdir(parents=True)
            (root / "docs" / "runbooks").mkdir(parents=True)
            (root / ".env.example").write_text(
                "LOCAL_ADMIN_USERNAME=admin\n"
                "LOCAL_ADMIN_PASSWORD=LocalDemoAdmin#12345\n"
                "LOCAL_SUPER_ADMIN_USERNAME=superadmin\n"
                "LOCAL_SUPER_ADMIN_PASSWORD=LocalDemoAdmin#12345\n"
                "UV_DEFAULT_INDEX=https://pypi.org/simple\n"
                "GOPROXY=https://proxy.golang.org,direct\n"
                "GOSUMDB=sum.golang.org\n"
                "# POSTGRES_IMAGE=docker.1ms.run/library/postgres:16-alpine\n"
                "# REDIS_IMAGE=docker.1ms.run/library/redis:7-alpine\n"
                "# MINIO_IMAGE=docker.1ms.run/minio/minio:RELEASE.2025-09-07T16-13-09Z\n"
                "# MINIO_MC_IMAGE=docker.1ms.run/minio/mc:RELEASE.2025-08-13T08-35-41Z\n"
                "VENDOR_RUNTIME_URL=http://127.0.0.1:9380\n"
                "VENDOR_RUNTIME_SERVICE_TOKEN=local-dev-runtime-service-token-change-me\n"
                "KNOWLEDGE_RUNTIME_SERVICE_TOKEN=local-dev-runtime-service-token-change-me\n"
                "KNOWLEDGE_RUNTIME_READINESS_MODE=query\n"
                "KNOWLEDGE_AUTO_START_INGESTION=true\n"
                "KNOWLEDGE_RUNTIME_WORKER_IDLE_SHUTDOWN_SECONDS=300\n"
                "KNOWLEDGE_RUNTIME_WORKER_IDLE_CHECK_SECONDS=15\n"
                "DOC_ENGINE=elasticsearch\n",
                encoding="utf-8",
            )
            (root / ".gitignore").write_text("/.local/\n*.pid\n", encoding="utf-8")
            (root / "services" / "parser").mkdir(parents=True)
            (root / "services" / "parser" / "uv.lock").write_text(
                'source = { registry = "https://pypi.tuna.tsinghua.edu.cn/simple" }\n'
                "https://pypi.tuna.tsinghua.edu.cn/packages/example.whl\n",
                encoding="utf-8",
            )
            (root / "deploy" / "seeds" / "001-local-demo-seed.sql").write_text(
                "\\connect auth_system\nINSERT INTO auth_users (id) VALUES ('usr_local_admin') ON CONFLICT (id) DO NOTHING;\n",
                encoding="utf-8",
            )
            (root / "deploy" / "seeds" / "002-ai-gateway-model-profiles.sql").write_text(
                "default-chat\nhttp://localhost:11434/v1\n",
                encoding="utf-8",
            )
            (root / "deploy" / "seeds" / "003-qa-document-mcp.sql").write_text(
                "\\connect qa_system\n"
                "INSERT INTO mcp_servers (alias) VALUES ('document') ON CONFLICT (alias) DO UPDATE;\n",
                encoding="utf-8",
            )
            (root / "deploy" / "seeds" / "004-qa-default-knowledge-base.sql").write_text(
                "\\connect qa_system\n"
                "keep QA's default knowledge-base list empty\n"
                "defaultKnowledgeBaseIds\n"
                "search all indexed\n"
                "DELETE FROM qa_config_knowledge_bases WHERE external_kb_id = 'kb_local_demo';\n",
                encoding="utf-8",
            )
            (root / "deploy" / "README.md").write_text(
                "configuration authority\n"
                "config/base.yaml\n"
                "config/dev.yaml\n"
                ".env.local\n"
                "cp .env.example .env.local\n"
                "LOCAL_ADMIN_USERNAME=admin\n"
                "LOCAL_ADMIN_PASSWORD=LocalDemoAdmin#12345\n"
                "LOCAL_SUPER_ADMIN_USERNAME=superadmin\n"
                "LOCAL_SUPER_ADMIN_PASSWORD=LocalDemoAdmin#12345\n"
                "admin / LocalDemoAdmin#12345\n"
                "superadmin / LocalDemoAdmin#12345\n"
                "Go modules 下载默认读取 profile\n"
                "源选择采用官方默认源\n"
                "active 第三方镜像值\n"
                "默认使用官方源\n"
                "--china\n"
                "大陆镜像\n"
                "GOPROXY=https://proxy.golang.org,direct\n"
                "GOSUMDB=sum.golang.org\n"
                "./scripts/local/check.sh\n"
                "./scripts/local/start.sh\n"
                "./scripts/local/stop.sh\n"
                "cleanup with down -v\n",
                encoding="utf-8",
            )
            (root / "config").mkdir(parents=True)
            (root / "config" / "README.md").write_text(
                "configuration authority\nconfig/base.yaml\nconfig/dev.yaml\n.env.local\n",
                encoding="utf-8",
            )
            (root / "config" / "base.yaml").write_text(
                "COMPOSE_PROJECT_NAME:\n"
                "POSTGRES_IMAGE:\nvalue: postgres:16-alpine\n"
                "REDIS_IMAGE:\nvalue: redis:7-alpine\n"
                "MINIO_IMAGE:\nvalue: minio/minio:RELEASE.2025-09-07T16-13-09Z\n"
                "MINIO_MC_IMAGE:\nvalue: minio/mc:RELEASE.2025-08-13T08-35-41Z\n",
                encoding="utf-8",
            )
            (root / "scripts" / "local").mkdir(parents=True)
            (root / "scripts" / "local" / "lib").mkdir(parents=True)
            (root / "scripts" / "local" / "lib" / "common.sh").write_text(
                "to_lower\n"
                "normalize_http_url\n"
                "append_no_proxy\n"
                "HF_ENDPOINT=https://hf-mirror.com\n",
                encoding="utf-8",
            )
            (root / "scripts" / "local" / "lib" / "process.sh").write_text(
                "setsid\n"
                "os.setsid()\n"
                'kill -0 -- "-$pid"\n'
                'kill -TERM -- "-$pid"\n'
                'kill -KILL -- "-$pid"\n',
                encoding="utf-8",
            )
            (root / "scripts" / "local" / "lib" / "knowledge-runtime.sh").write_text(
                ".local/knowledge-runtime/service_conf.yaml\n",
                encoding="utf-8",
            )
            (root / "scripts" / "local" / "check.sh").write_text(
                "[check]\n"
                "no downloads or builds will run\n"
                "setup suggestions\n"
                "Official sources, run manually only for missing items\n"
                "Mainland China mirrors, run manually only for missing items\n"
                "--sync-only --profile\n"
                "ragflow_deps/download_deps.py --skip-uv-sync\n"
                "docker.1ms.run/library/postgres:16-alpine\n"
                "goose@v3.27.1\n"
                "https://go.dev/dl/\n"
                "https://docs.astral.sh/uv\n",
                encoding="utf-8",
            )
            (root / "scripts" / "local" / "render_ai_gateway_local_seed.go").write_text(
                "package main\n",
                encoding="utf-8",
            )
            (root / "scripts" / "local" / "start.sh").write_text(
                "[start]\n"
                "This script does not run dependency downloads\n"
                "--pull never\n"
                ".local/tools/config-ctl\n"
                ".local/tools/goose\n"
                ".local/bin\n"
                "AUTH_DATABASE_URL\n"
                "FILE_DATABASE_URL\n"
                "KNOWLEDGE_DATABASE_URL\n"
                "QA_DATABASE_URL\n"
                "DOCUMENT_DATABASE_URL\n"
                "AI_GATEWAY_DATABASE_URL\n"
                "001-local-demo-seed.sql\n"
                "002-ai-gateway-model-profiles.sql\n"
                "003-qa-document-mcp.sql\n"
                "004-qa-default-knowledge-base.sql\n"
                "render-ai-gateway-local-seed\n"
                "--runtime api\n"
                "--runtime full\n"
                "knowledge-runtime-api\n"
                "knowledge-runtime-worker\n"
                "go mod download\n"
                "go run module@version\n",
                encoding="utf-8",
            )
            (root / "scripts" / "local" / "clean.sh").write_text(
                "[clean]\n"
                "./scripts/local/stop.sh\n"
                "down -v\n"
                "--remove-orphans\n"
                ".local/tools/config-ctl\n"
                "Images, source files, .env.local, .local/tools, and .local/bin are not removed.\n",
                encoding="utf-8",
            )
            (root / "scripts" / "local" / "stop.sh").write_text(
                "[stop]\n"
                "[ok]\n"
                "[warn]\n"
                "[fail]\n"
                "[hint]\n"
                "completed successfully\n"
                "failed during\n"
                "nothing to stop\n"
                'kill -0 -- "-$pid"\n'
                'kill -TERM -- "-$pid"\n'
                'kill -KILL -- "-$pid"\n',
                encoding="utf-8",
            )
            (root / "services" / "ai-gateway" / "cmd" / "local-seed").mkdir(parents=True)
            (root / "services" / "ai-gateway" / "cmd" / "local-seed" / "main.go").write_text(
                "package main\n",
                encoding="utf-8",
            )
            (root / "docs" / "runbooks" / "local-integration.md").write_text(
                "local integration local seed\n",
                encoding="utf-8",
            )

            issues = verifier.verify_local_seed_contract(root)

        self.assertIssueContains(issues, "doc_local_demo_seed")
        self.assertIssueContains(issues, "22222222-2222-4222-8222-222222222301")
        self.assertIssueContains(issues, "33333333-3333-4333-8333-333333333301")
        self.assertIssueContains(issues, "AUTH_DATABASE_URL")

    def test_verifier_reports_missing_runtime_env_defaults(self) -> None:
        verifier = load_verifier()

        issues = verifier.validate_docs(
            deploy_readme="唯一默认配置来源\n",
            runbook="",
            env_example="VENDOR_RUNTIME_URL=http://127.0.0.1:9380\n",
            config_readme="",
            config_base="",
            check_script="",
            start_script="",
            clean_script="",
            ai_gateway_local_seed_renderer="",
            stop_script="",
            ai_gateway_local_seed_main="",
        )

        self.assertIssueContains(issues, "DOC_ENGINE:")
        self.assertIssueContains(issues, ".local/bin")

    def test_verifier_reports_missing_local_runtime_gitignore(self) -> None:
        verifier = load_verifier()

        issues = verifier.validate_gitignore("*.pid\n*.log\n")

        self.assertIssueContains(issues, ".gitignore")
        self.assertIssueContains(issues, "/.local/")

    def test_verifier_reports_container_only_ai_gateway_seed_url(self) -> None:
        verifier = load_verifier()

        issues = verifier.validate_seed_002(
            """
            default-chat
            default-embedding
            default-rerank
            host.docker.internal
            cred-local-chat
            cred-local-embedding
            cred-local-rerank
            local-demo-key-v1
            ON CONFLICT
            ON CONFLICT
            """
        )

        self.assertIssueContains(issues, "host.docker.internal")
        self.assertIssueContains(issues, "http://localhost:11434/v1")

    def test_verifier_reports_incomplete_document_mcp_seed(self) -> None:
        verifier = load_verifier()

        issues = verifier.validate_seed_003(
            """
            \\connect qa_system
            INSERT INTO mcp_servers (alias, token_encrypted)
            VALUES ('document', NULL)
            ON CONFLICT (alias) DO UPDATE;
            """
        )

        self.assertIssueContains(issues, "http://localhost:8085/mcp")
        self.assertIssueContains(issues, "33333333-3333-4333-8333-333333333601")

    def test_verifier_reports_missing_auth_qa_settings_permissions(self) -> None:
        verifier = load_verifier()

        issues = verifier.validate_auth_migrations(
            """
            INSERT INTO auth_permissions (code) VALUES ('qa:use');
            INSERT INTO role_permissions (id) VALUES ('rperm_admin_qa_use');
            """
        )

        self.assertIssueContains(issues, "qa:settings:read")
        self.assertIssueContains(issues, "qa:settings:write")
        self.assertIssueContains(issues, "rperm_admin_qa_settings_read")
        self.assertIssueContains(issues, "rperm_super_qa_settings_write")

    def assertIssueContains(self, issues: list[str], expected: str) -> None:
        self.assertTrue(
            any(expected in issue for issue in issues),
            f"Expected issue containing {expected!r}, got: {issues!r}",
        )


if __name__ == "__main__":
    unittest.main()
