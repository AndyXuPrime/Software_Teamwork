import os
import shutil
import stat
import subprocess
import textwrap
import tempfile
import unittest
from pathlib import Path
from typing import Optional


class CleanScriptTests(unittest.TestCase):
    def test_clean_with_yes_stops_processes_and_removes_volumes(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = self.prepare_runtime(Path(directory))

            result = self.run_clean(root, args=["--yes"])

            self.assertEqual(0, result.returncode, result.stderr)
            self.assertIn("local development data cleared", result.stdout)
            sequence = (root / "sequence.log").read_text(encoding="utf-8")
            self.assertIn("stop\n", sequence)
            self.assertIn("docker:compose -f", sequence)
            self.assertIn(" config --quiet\n", sequence)
            self.assertIn(" down -v --remove-orphans\n", sequence)
            self.assertLess(sequence.index("stop"), sequence.index(" down -v --remove-orphans"))

    def test_non_interactive_clean_requires_yes(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = self.prepare_runtime(Path(directory))

            result = self.run_clean(root)

            self.assertEqual(2, result.returncode)
            self.assertIn("refusing to clean without --yes", result.stderr)
            self.assertFalse((root / "sequence.log").exists())

    def test_can_skip_stop_and_keep_orphans(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = self.prepare_runtime(Path(directory))

            result = self.run_clean(root, args=["--yes", "--skip-stop", "--keep-orphans"])

            self.assertEqual(0, result.returncode, result.stderr)
            sequence = (root / "sequence.log").read_text(encoding="utf-8")
            self.assertNotIn("stop", sequence)
            self.assertIn(" down -v\n", sequence)
            self.assertNotIn("--remove-orphans", sequence)

    def prepare_runtime(self, root: Path) -> Path:
        for relative in [
            "scripts/local/clean.sh",
            "scripts/config/load-profile.sh",
        ]:
            source = Path.cwd() / relative
            target = root / relative
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source, target)
            target.chmod(target.stat().st_mode | stat.S_IXUSR)

        self.write_executable(
            root / "scripts" / "local" / "stop.sh",
            "#!/usr/bin/env bash\necho stop >> \"$FAKE_SEQUENCE\"\n",
        )
        self.write_executable(
            root / ".local" / "tools" / "config-ctl",
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

        (root / "deploy").mkdir()
        (root / "deploy" / "docker-compose.yml").write_text("services: {}\n", encoding="utf-8")
        (root / ".env.local").write_text("POSTGRES_ADMIN_URL=postgres://example/postgres\n", encoding="utf-8")

        fake_bin = root / "fake-bin"
        fake_bin.mkdir()
        for command in ["bash", "cp", "dirname", "mkdir"]:
            target = shutil.which(command)
            if target is None:
                raise AssertionError(f"{command} is required to run clean.sh tests")
            os.symlink(target, fake_bin / command)
        self.write_executable(
            fake_bin / "docker",
            "#!/usr/bin/env bash\necho \"docker:$*\" >> \"$FAKE_SEQUENCE\"\nexit 0\n",
        )
        return root

    def run_clean(
        self,
        root: Path,
        args: Optional[list[str]] = None,
    ) -> subprocess.CompletedProcess[str]:
        env = os.environ.copy()
        env["FAKE_ROOT"] = str(root)
        env["FAKE_SEQUENCE"] = str(root / "sequence.log")
        env["PATH"] = f"{root / 'fake-bin'}{os.pathsep}{env['PATH']}"
        return subprocess.run(
            [str(root / "scripts" / "local" / "clean.sh"), *(args or [])],
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


if __name__ == "__main__":
    unittest.main()
