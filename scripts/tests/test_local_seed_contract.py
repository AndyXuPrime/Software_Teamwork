import importlib
import tempfile
import textwrap
import unittest
from pathlib import Path


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

    def test_verifier_reports_missing_required_resource_ids(self) -> None:
        verifier = load_verifier()
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "deploy" / "seeds").mkdir(parents=True)
            (root / "docs" / "runbooks").mkdir(parents=True)
            (root / "deploy" / ".env.example").write_text(
                "LOCAL_ADMIN_USERNAME=admin\n"
                "LOCAL_ADMIN_PASSWORD=LocalDemoAdmin#12345\n"
                "UV_DEFAULT_INDEX=https://pypi.tuna.tsinghua.edu.cn/simple\n",
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
            (root / "deploy" / "README.md").write_text(
                "deploy/.env.example 是唯一默认配置来源\n"
                "cp deploy/.env.example deploy/.env\n"
                "./scripts/local/dev-up.sh\n"
                "./scripts/local/run-backend.sh\n"
                "LOCAL_ADMIN_USERNAME=admin\n"
                "LOCAL_ADMIN_PASSWORD=LocalDemoAdmin#12345\n"
                "admin / LocalDemoAdmin#12345\n"
                "cleanup with down -v\n",
                encoding="utf-8",
            )
            (root / "scripts" / "local").mkdir(parents=True)
            (root / "scripts" / "local" / "dev-up.sh").write_text(
                "goose@v3.27.1\n"
                "psql\n"
                "001-local-demo-seed.sql\n"
                "--wait\n"
                "--wait-timeout\n"
                "initialize_qdrant_collection\n"
                "QDRANT_URL\n"
                "QDRANT_COLLECTION\n"
                "EMBEDDING_DIMENSION\n"
                "Cosine\n",
                encoding="utf-8",
            )
            (root / "scripts" / "local" / "run-backend.sh").write_text(
                "setsid\n"
                "uv sync --frozen --group dev --extra paddleocr\n"
                "uv run --frozen parser-service\n"
                "auth\nfile\nparser\nknowledge\nai-gateway\nqa\ndocument\ngateway\n",
                encoding="utf-8",
            )
            (root / "scripts" / "local" / "stop-backend.sh").write_text(
                'kill -0 -- "-$pid"\n'
                'kill -TERM -- "-$pid"\n'
                'kill -KILL -- "-$pid"\n',
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

    def test_verifier_reports_parser_lock_official_pypi_urls(self) -> None:
        verifier = load_verifier()

        issues = verifier.validate_parser_uv_lock(
            'source = { registry = "https://pypi.org/simple" }\n'
            "https://files.pythonhosted.org/packages/example.whl\n"
        )

        self.assertIssueContains(issues, "https://pypi.org/simple")
        self.assertIssueContains(issues, "https://files.pythonhosted.org")

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
