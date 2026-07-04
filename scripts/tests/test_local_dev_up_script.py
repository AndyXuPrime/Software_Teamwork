import os
import signal
import shutil
import stat
import subprocess
import textwrap
import tempfile
import time
import unittest
from pathlib import Path
from typing import Optional


class LocalStartupScriptTests(unittest.TestCase):
    def test_check_default_only_inspects_environment(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = self.prepare_runtime(Path(directory))
            self.create_prepared_config_tool(root)
            self.create_prepared_tools(root)
            self.create_service_binaries(root)
            self.create_runtime_files(root)

            result = self.run_check(root)

            self.assertEqual(0, result.returncode, result.stderr)
            self.assertIn("checking local environment; no downloads or builds will run", result.stdout)
            self.assertIn("local environment looks ready", result.stdout)
            self.assertFalse((root / "go-calls.log").exists())
            docker_calls = (root / "docker-calls.log").read_text(encoding="utf-8")
            self.assertNotIn("image inspect", docker_calls)
            self.assertNotIn(" pull ", f" {docker_calls} ")
            self.assertFalse((root / "uv-calls.log").exists())

    def test_check_china_mode_prints_manual_download_suggestions(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = self.prepare_runtime(Path(directory))

            result = self.run_check(root, args=["--china"])

            self.assertNotEqual(0, result.returncode)
            self.assertIn("Mainland China mirrors, run manually only for missing items", result.stdout)
            self.assertIn("docker pull docker.1ms.run/library/postgres:16-alpine", result.stdout)
            self.assertIn("GOPROXY=https://goproxy.cn,direct", result.stdout)
            self.assertIn("ragflow_deps/download_deps.py --skip-uv-sync --china", result.stdout)
            self.assertFalse((root / "go-calls.log").exists())
            if (root / "docker-calls.log").exists():
                docker_calls = (root / "docker-calls.log").read_text(encoding="utf-8")
                self.assertNotIn("image inspect", docker_calls)
                self.assertNotIn(" pull ", f" {docker_calls} ")
            self.assertFalse((root / "uv-calls.log").exists())

    def test_start_infra_only_never_pulls_or_requires_uv(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = self.prepare_runtime(Path(directory))
            self.create_prepared_config_tool(root)
            self.create_prepared_tools(root)
            (root / "fake-bin" / "uv").unlink()

            result = self.run_start(root, args=["--infra-only"])

            self.assertEqual(0, result.returncode, result.stderr)
            docker_calls = (root / "docker-calls.log").read_text(encoding="utf-8")
            self.assertIn("up -d --pull never --wait", docker_calls)
            self.assertIn("up --no-deps --pull never --exit-code-from minio-init minio-init", docker_calls)
            self.assertIn("ps postgres redis minio elasticsearch", docker_calls)
            self.assertNotIn(" pull ", f" {docker_calls} ")
            self.assertFalse((root / "uv-calls.log").exists())
            self.assertFalse((root / "go-calls.log").exists())

    def test_start_china_rewrites_rendered_compose_env_for_infra(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = self.prepare_runtime(Path(directory))
            self.create_prepared_config_tool(root)
            self.create_prepared_tools(root)

            result = self.run_start(root, args=["--infra-only", "--china"])

            self.assertEqual(0, result.returncode, result.stderr)
            compose_env = (root / ".local" / "config" / "dev.env").read_text(encoding="utf-8")
            self.assertIn("POSTGRES_IMAGE=docker.1ms.run/library/postgres:16-alpine", compose_env)
            self.assertIn("REDIS_IMAGE=docker.1ms.run/library/redis:7-alpine", compose_env)
            self.assertIn("MINIO_IMAGE=docker.1ms.run/minio/minio:RELEASE.2025-09-07T16-13-09Z", compose_env)
            self.assertIn(
                "KNOWLEDGE_RUNTIME_ELASTICSEARCH_IMAGE=docker.1ms.run/elasticsearch:8.15.3",
                compose_env,
            )
            docker_env = (root / "docker-env.log").read_text(encoding="utf-8")
            self.assertIn("POSTGRES_IMAGE=docker.1ms.run/library/postgres:16-alpine", docker_env)
            self.assertIn("KNOWLEDGE_RUNTIME_ELASTICSEARCH_IMAGE=docker.1ms.run/elasticsearch:8.15.3", docker_env)
            self.assertNotIn("docker.1ms.run", (root / ".env.local").read_text(encoding="utf-8"))

    def test_start_fails_with_hint_when_service_binaries_are_missing(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = self.prepare_runtime(Path(directory))
            self.create_prepared_config_tool(root)
            self.create_prepared_tools(root)

            result = self.run_start(root, args=["--backend-only", "--no-runtime"])

            self.assertNotEqual(0, result.returncode)
            self.assertIn("missing", result.stderr)
            self.assertIn(".local/bin/auth-server", result.stderr)
            self.assertIn("./scripts/local/check.sh", result.stderr)
            self.assertFalse((root / "docker-calls.log").exists())

    def test_start_backend_uses_prepared_binaries_without_runtime(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = self.prepare_runtime(Path(directory))
            self.create_prepared_config_tool(root)
            self.create_prepared_tools(root)
            self.create_service_binaries(root)

            try:
                result = self.run_start(
                    root,
                    args=["--backend-only", "--no-runtime"],
                    extra_env={"LOCAL_STARTUP_CHECK_SECONDS": "0"},
                )

                self.assertEqual(0, result.returncode, result.stderr)
                self.assertIn("local backend and runtime mode 'none' are running", result.stdout)
                self.assertIn("host process groups", result.stdout)
                for service in ["auth", "file", "knowledge", "ai-gateway", "qa", "document", "gateway"]:
                    self.assertTrue((root / ".local" / "run" / f"{service}.pid").exists())
                service_env = (root / "service-env.log").read_text(encoding="utf-8")
                self.assertIn(f"auth-server PWD={root / 'services' / 'auth'}", service_env)
                self.assertIn(f"qa-server PWD={root / 'services' / 'qa'}", service_env)
                self.assertNotIn(f"qa-server PWD={root} ", service_env)
                self.assertFalse((root / "go-calls.log").exists())
                self.assertFalse((root / "uv-calls.log").exists())
            finally:
                self.cleanup_started_processes(root)

    def test_start_aligns_placeholder_model_env_with_local_chat_model(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = self.prepare_runtime(Path(directory))
            self.create_prepared_config_tool(root)
            self.create_prepared_tools(root)
            self.create_service_binaries(root)
            with (root / ".env.local").open("a", encoding="utf-8") as env:
                env.write(
                    "AI_GATEWAY_LOCAL_CHAT_MODEL=deepseek-ai/DeepSeek-V4-Flash\n"
                    "MODEL_ID=local-placeholder-chat\n"
                    "DOCUMENT_AI_GATEWAY_MODEL=local-placeholder-chat\n"
                )

            try:
                result = self.run_start(
                    root,
                    args=["--backend-only", "--no-runtime"],
                    extra_env={"LOCAL_STARTUP_CHECK_SECONDS": "0"},
                )

                self.assertEqual(0, result.returncode, result.stderr)
                self.assertIn("using AI_GATEWAY_LOCAL_CHAT_MODEL for host-run QA MODEL_ID", result.stdout)
                self.assertIn("using AI_GATEWAY_LOCAL_CHAT_MODEL for host-run Document model", result.stdout)
                service_env = (root / "service-env.log").read_text(encoding="utf-8")
                self.assertIn("MODEL_ID=deepseek-ai/DeepSeek-V4-Flash", service_env)
                self.assertIn("DOCUMENT_AI_GATEWAY_MODEL=deepseek-ai/DeepSeek-V4-Flash", service_env)
            finally:
                self.cleanup_started_processes(root)

    def test_start_defaults_to_prepared_runtime_worker(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = self.prepare_runtime(Path(directory))
            self.create_prepared_config_tool(root)
            self.create_prepared_tools(root)
            self.create_service_binaries(root)
            self.create_runtime_files(root)

            try:
                result = self.run_start(root, args=["--backend-only"], extra_env={"LOCAL_STARTUP_CHECK_SECONDS": "0"})

                self.assertEqual(0, result.returncode, result.stderr)
                self.assertIn("runtime mode 'full' are running", result.stdout)
                self.assertIn("host process groups", result.stdout)
                self.assertTrue((root / ".local" / "run" / "knowledge-runtime-api.pid").exists())
                self.assertTrue((root / ".local" / "run" / "knowledge-runtime-worker.pid").exists())
                self.assertFalse((root / "go-calls.log").exists())
                self.assertFalse((root / "uv-calls.log").exists())
            finally:
                self.cleanup_started_processes(root)

    def prepare_runtime(self, root: Path) -> Path:
        for relative in [
            "scripts/local/check.sh",
            "scripts/local/start.sh",
            "scripts/config/load-profile.sh",
            "scripts/local/lib/common.sh",
            "scripts/local/lib/process.sh",
            "scripts/local/lib/knowledge-runtime.sh",
        ]:
            source = Path.cwd() / relative
            target = root / relative
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source, target)
            target.chmod(target.stat().st_mode | stat.S_IXUSR)

        (root / "deploy" / "seeds").mkdir(parents=True)
        (root / "deploy" / "docker-compose.yml").write_text("services: {}\n", encoding="utf-8")
        (root / ".env.local").write_text(
            textwrap.dedent(
                """\
                UV_DEFAULT_INDEX=https://pypi.org/simple
                GOPROXY=https://proxy.golang.org,direct
                GOSUMDB=sum.golang.org
                AUTH_DATABASE_URL=postgres://example/auth
                FILE_DATABASE_URL=postgres://example/file
                KNOWLEDGE_DATABASE_URL=postgres://example/knowledge
                QA_DATABASE_URL=postgres://example/qa
                DOCUMENT_DATABASE_URL=postgres://example/document
                AI_GATEWAY_DATABASE_URL=postgres://example/ai_gateway
                POSTGRES_ADMIN_URL=postgres://example/postgres
                """
            ),
            encoding="utf-8",
        )
        for seed in [
            "001-local-demo-seed.sql",
            "002-ai-gateway-model-profiles.sql",
            "003-qa-document-mcp.sql",
            "004-qa-default-knowledge-base.sql",
        ]:
            (root / "deploy" / "seeds" / seed).write_text("-- seed\n", encoding="utf-8")
        for service in ["auth", "file", "knowledge", "ai-gateway", "qa", "document", "gateway"]:
            (root / "services" / service).mkdir(parents=True)
        (root / "services" / "knowledge-runtime" / "ragflow_deps").mkdir(parents=True)
        (root / "services" / "knowledge-runtime" / "rag").mkdir(parents=True)
        (root / "config" / "ctl").mkdir(parents=True)

        fake_bin = root / "fake-bin"
        fake_bin.mkdir()
        for command in ["bash", "cat", "cp", "dirname", "mkdir", "sleep"]:
            target = shutil.which(command)
            if target is None:
                raise AssertionError(f"{command} is required to run local script tests")
            os.symlink(target, fake_bin / command)
        self.write_executable(
            fake_bin / "docker",
            """\
            #!/usr/bin/env bash
            echo "$*" >> "$FAKE_DOCKER_CALLS"
            {
              echo "POSTGRES_IMAGE=${POSTGRES_IMAGE:-}"
              echo "REDIS_IMAGE=${REDIS_IMAGE:-}"
              echo "MINIO_IMAGE=${MINIO_IMAGE:-}"
              echo "MINIO_MC_IMAGE=${MINIO_MC_IMAGE:-}"
              echo "KNOWLEDGE_RUNTIME_ELASTICSEARCH_IMAGE=${KNOWLEDGE_RUNTIME_ELASTICSEARCH_IMAGE:-}"
            } >> "$FAKE_DOCKER_ENV"
            exit 0
            """,
        )
        self.write_executable(
            fake_bin / "go",
            """\
            #!/usr/bin/env bash
            echo "$PWD|$*" >> "$FAKE_GO_CALLS"
            {
              echo "GOPROXY=${GOPROXY:-}"
              echo "GOSUMDB=${GOSUMDB:-}"
            } >> "$FAKE_GO_ENV"
            exit 0
            """,
        )
        self.write_executable(fake_bin / "psql", "#!/usr/bin/env bash\ncat >/dev/null || true\nexit 0\n")
        self.write_executable(fake_bin / "bun", "#!/usr/bin/env bash\nexit 0\n")
        self.write_executable(fake_bin / "python3", "#!/usr/bin/env bash\nexit 0\n")
        self.write_executable(fake_bin / "curl", "#!/usr/bin/env bash\nprintf '200'\nexit 0\n")
        self.write_executable(fake_bin / "uv", "#!/usr/bin/env bash\necho \"$*\" >> \"$FAKE_UV_CALLS\"\nexit 0\n")
        self.write_executable(fake_bin / "git", "#!/usr/bin/env bash\necho 0123456789abcdef0123456789abcdef01234567\n")
        self.write_executable(
            fake_bin / "tail",
            "#!/usr/bin/env bash\nexit 0\n",
        )
        return root

    def create_prepared_config_tool(self, root: Path) -> None:
        tool = root / ".local" / "tools" / "config-ctl"
        self.write_executable(
            tool,
            """\
            #!/usr/bin/env bash
            format=dotenv
            out=
            while (($# > 0)); do
              case "$1" in
                --format) format="$2"; shift 2 ;;
                --out) out="$2"; shift 2 ;;
                *) shift ;;
              esac
            done
            mkdir -p "$(dirname "$out")"
            if [[ "$format" == "shell" ]]; then
              while IFS= read -r line || [[ -n "$line" ]]; do
                [[ -z "$line" || "$line" =~ ^[[:space:]]*# || "$line" != *=* ]] && continue
                key="${line%%=*}"
                value="${line#*=}"
                printf "export %s=%q\\n" "$key" "$value"
              done < "$FAKE_ROOT/.env.local" > "$out"
            else
              cp "$FAKE_ROOT/.env.local" "$out"
            fi
            """,
        )

    def create_prepared_tools(self, root: Path) -> None:
        self.write_executable(root / ".local" / "tools" / "goose", "#!/usr/bin/env bash\nexit 0\n")
        self.write_executable(
            root / ".local" / "tools" / "render-ai-gateway-local-seed",
            "#!/usr/bin/env bash\nexit 0\n",
        )

    def create_service_binaries(self, root: Path) -> None:
        for binary in [
            "auth-server",
            "file-server",
            "knowledge-adapter",
            "ai-gateway-server",
            "qa-server",
            "document-server",
            "gateway-server",
        ]:
            self.write_executable(
                root / ".local" / "bin" / binary,
                """\
                #!/usr/bin/env bash
                printf '%s PWD=%s MODEL_ID=%s DOCUMENT_AI_GATEWAY_MODEL=%s\\n' \
                  "$(basename "$0")" "$PWD" "${MODEL_ID:-}" "${DOCUMENT_AI_GATEWAY_MODEL:-}" >> "$FAKE_ROOT/service-env.log"
                while true; do sleep 60; done
                """,
            )

    def create_runtime_files(self, root: Path) -> None:
        runtime = root / "services" / "knowledge-runtime"
        (runtime / ".venv").mkdir(parents=True)
        (runtime / "ragflow_deps" / "nltk_data").mkdir(parents=True)
        (runtime / "rag" / "res" / "deepdoc").mkdir(parents=True)
        (runtime / "ragflow_deps" / "tika-server-standard-3.3.0.jar").write_text("jar\n", encoding="utf-8")
        (runtime / "ragflow_deps" / "cl100k_base.tiktoken").write_text("encoding\n", encoding="utf-8")
        self.write_executable(
            runtime / "deploy" / "api" / "run-local.sh",
            "#!/usr/bin/env bash\nwhile true; do sleep 60; done\n",
        )
        self.write_executable(
            runtime / "deploy" / "worker" / "run-local.sh",
            "#!/usr/bin/env bash\nwhile true; do sleep 60; done\n",
        )
        (runtime / "conf").mkdir(parents=True)
        (runtime / "conf" / "service_conf.yaml").write_text(
            "es:\n  hosts: http://127.0.0.1:9200\n",
            encoding="utf-8",
        )

    def run_check(
        self,
        root: Path,
        args: Optional[list[str]] = None,
        extra_env: Optional[dict[str, str]] = None,
    ) -> subprocess.CompletedProcess[str]:
        return self.run_script(root, "check.sh", args, extra_env)

    def run_start(
        self,
        root: Path,
        args: Optional[list[str]] = None,
        extra_env: Optional[dict[str, str]] = None,
    ) -> subprocess.CompletedProcess[str]:
        return self.run_script(root, "start.sh", args, extra_env)

    def run_script(
        self,
        root: Path,
        script: str,
        args: Optional[list[str]] = None,
        extra_env: Optional[dict[str, str]] = None,
    ) -> subprocess.CompletedProcess[str]:
        env = os.environ.copy()
        env.update(extra_env or {})
        env["FAKE_ROOT"] = str(root)
        env["FAKE_DOCKER_CALLS"] = str(root / "docker-calls.log")
        env["FAKE_DOCKER_ENV"] = str(root / "docker-env.log")
        env["FAKE_GO_CALLS"] = str(root / "go-calls.log")
        env["FAKE_GO_ENV"] = str(root / "go-env.log")
        env["FAKE_UV_CALLS"] = str(root / "uv-calls.log")
        env["PATH"] = f"{root / 'fake-bin'}{os.pathsep}{env['PATH']}"
        return subprocess.run(
            [str(root / "scripts" / "local" / script), *(args or [])],
            cwd=root,
            env=env,
            text=True,
            capture_output=True,
            check=False,
        )

    def write_executable(self, path: Path, content: str) -> None:
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(textwrap.dedent(content), encoding="utf-8")
        path.chmod(path.stat().st_mode | stat.S_IXUSR)

    def cleanup_started_processes(self, root: Path) -> None:
        run_dir = root / ".local" / "run"
        if not run_dir.exists():
            return
        pids: list[int] = []
        for pid_file in run_dir.glob("*.pid"):
            try:
                pids.append(int(pid_file.read_text(encoding="utf-8").strip()))
            except ValueError:
                continue
        for pid in pids:
            try:
                os.kill(-pid, signal.SIGTERM)
            except ProcessLookupError:
                continue
            except PermissionError:
                continue
        time.sleep(0.1)
        for pid in pids:
            try:
                os.kill(-pid, 0)
            except ProcessLookupError:
                continue
            try:
                os.kill(-pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            except PermissionError:
                pass


if __name__ == "__main__":
    unittest.main()
