import unittest
from pathlib import Path


class KnowledgeRuntimeDependencySplitTests(unittest.TestCase):
    def test_start_helper_prepares_runtime_dependencies_before_execution(self) -> None:
        script = Path("scripts/local/start.sh").read_text(encoding="utf-8")

        self.assertIn("--runtime api", script)
        self.assertIn("--runtime full", script)
        self.assertIn("prepare_runtime_dependencies()", script)
        self.assertIn("--sync-only", script)
        self.assertIn("--profile", script)
        self.assertIn("download_deps.py", script)
        self.assertIn("--skip-uv-sync", script)
        self.assertIn(".local-start-profile", script)
        self.assertIn("runtime_dependencies_synced", script)
        self.assertIn("mark_runtime_dependencies_synced", script)
        self.assertIn('start_process "knowledge-runtime-api"', script)
        self.assertIn('start_process "knowledge-runtime-worker"', script)
        self.assertNotIn("\nuv sync", script)
        self.assertNotIn("exec uv sync", script)

    def test_runtime_entrypoints_use_no_sync_execution_only(self) -> None:
        api_script = Path("services/knowledge-runtime/deploy/api/run-local.sh").read_text(encoding="utf-8")
        worker_script = Path("services/knowledge-runtime/deploy/worker/run-local.sh").read_text(encoding="utf-8")

        self.assertIn("uv run --no-sync --no-default-groups", api_script)
        self.assertIn("uv run --no-sync --group worker", worker_script)
        self.assertNotIn("\nuv sync", api_script)
        self.assertNotIn("exec uv sync", api_script)
        self.assertNotIn("\nuv sync", worker_script)
        self.assertNotIn("exec uv sync", worker_script)
        self.assertNotIn('HF_ENDPOINT="${HF_ENDPOINT:-https://hf-mirror.com}"', api_script)
        self.assertIn("KNOWLEDGE_RUNTIME_REQUIRE_NLTK_DATA", worker_script)
        self.assertIn("NLTK_DATA", worker_script)

    def test_download_deps_prefetches_models_to_worker_runtime_path(self) -> None:
        script = Path("services/knowledge-runtime/ragflow_deps/download_deps.py").read_text(encoding="utf-8")

        self.assertIn("model_local_directory", script)
        self.assertIn('"rag" / "res" / "deepdoc"', script)
        self.assertIn("Falling back to direct file downloads", script)
        self.assertIn("HfApi", script)
        self.assertIn('"InfiniFlow/deepdoc"', script)
        self.assertIn('"InfiniFlow/text_concat_xgb_v1.0"', script)

    def test_china_runtime_deps_use_mozilla_compatible_ubuntu_mirror(self) -> None:
        script = Path("services/knowledge-runtime/ragflow_deps/download_deps.py").read_text(encoding="utf-8")

        self.assertIn("https://repo.huaweicloud.com/ubuntu/pool/main/o/openssl/libssl1.1_1.1.1f-1ubuntu2_amd64.deb", script)
        self.assertIn("https://repo.huaweicloud.com/ubuntu-ports/pool/main/o/openssl/libssl1.1_1.1.1f-1ubuntu2_arm64.deb", script)
        self.assertNotIn("mirrors.tuna.tsinghua.edu.cn/ubuntu/pool/main/o/openssl/libssl1.1", script)
        self.assertIn('opener.addheaders = [("User-Agent", "Mozilla/5.0")]', script)

    def test_download_deps_no_longer_downloads_uv_releases(self) -> None:
        script = Path("services/knowledge-runtime/ragflow_deps/download_deps.py").read_text(encoding="utf-8")

        self.assertNotIn("astral-sh/uv/releases/download", script)
        self.assertNotIn("uv-x86_64-unknown-linux-gnu.tar.gz", script)
        self.assertNotIn("uv-aarch64-unknown-linux-gnu.tar.gz", script)

    def test_download_deps_supports_sync_only_profiles(self) -> None:
        script = Path("services/knowledge-runtime/ragflow_deps/download_deps.py").read_text(encoding="utf-8")

        self.assertIn("--sync-only", script)
        self.assertIn("--profile", script)
        self.assertIn('"api"', script)
        self.assertIn('"worker"', script)
        self.assertIn('"all"', script)
        self.assertIn("sync_runtime_dependencies(args.china_mirrors, args.profile)", script)
        self.assertIn('"--no-default-groups"', script)
        self.assertIn('"--group"', script)
        self.assertIn('"worker"', script)
        self.assertIn('"--all-groups"', script)


if __name__ == "__main__":
    unittest.main()
