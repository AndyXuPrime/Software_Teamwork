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
                "LOCAL_ADMIN_USERNAME=admin\nLOCAL_ADMIN_PASSWORD=LocalDemoAdmin#12345\n",
                encoding="utf-8",
            )
            (root / "deploy" / "docker-compose.yml").write_text(
                textwrap.dedent(
                    """
                    services:
                      seed-local:
                        depends_on:
                          migrate-auth:
                            condition: service_completed_successfully
                    """
                ),
                encoding="utf-8",
            )
            (root / "deploy" / "seeds" / "001-local-demo-seed.sql").write_text(
                "\\connect auth_system\nINSERT INTO auth_users (id) VALUES ('usr_local_admin') ON CONFLICT (id) DO NOTHING;\n",
                encoding="utf-8",
            )
            (root / "deploy" / "seeds" / "002-ai-gateway-model-profiles.sql").write_text(
                "default-chat\n",
                encoding="utf-8",
            )
            (root / "deploy" / "README.md").write_text(
                "seed-local admin LocalDemoAdmin#12345\n",
                encoding="utf-8",
            )
            (root / "docs" / "runbooks" / "local-integration.md").write_text(
                "local integration seed-local\n",
                encoding="utf-8",
            )

            issues = verifier.verify_local_seed_contract(root)

        self.assertIssueContains(issues, "doc_local_demo_seed")
        self.assertIssueContains(issues, "22222222-2222-4222-8222-222222222301")
        self.assertIssueContains(issues, "33333333-3333-4333-8333-333333333301")
        self.assertIssueContains(issues, "migrate-qa")

    def assertIssueContains(self, issues: list[str], expected: str) -> None:
        self.assertTrue(
            any(expected in issue for issue in issues),
            f"Expected issue containing {expected!r}, got: {issues!r}",
        )


if __name__ == "__main__":
    unittest.main()
