import os
import shlex
import subprocess
import textwrap
import unittest
from pathlib import Path


class LocalCommonHelperTests(unittest.TestCase):
    def test_append_no_proxy_for_url_only_adds_loopback_hosts(self) -> None:
        helper = Path.cwd() / "scripts" / "local" / "lib" / "common.sh"
        script = textwrap.dedent(
            f"""\
            set -Eeuo pipefail
            . {shlex.quote(str(helper))}
            export NO_PROXY=example.internal
            append_no_proxy_for_url http://127.0.0.1:9380
            append_no_proxy_for_url https://pypi.org/simple
            append_no_proxy_for_url https://proxy.golang.org
            printf '%s\\n' "$NO_PROXY"
            if should_bypass_proxy_for_url http://localhost:9200; then
              echo local-bypass
            fi
            if should_bypass_proxy_for_url https://pypi.org/simple; then
              echo external-bypass
            else
              echo external-proxy
            fi
            """
        )

        result = subprocess.run(
            ["bash", "-c", script],
            cwd=Path.cwd(),
            env={**os.environ, "NO_COLOR": "1"},
            text=True,
            capture_output=True,
            check=False,
        )

        self.assertEqual(0, result.returncode, result.stderr)
        lines = result.stdout.splitlines()
        self.assertEqual("example.internal,127.0.0.1", lines[0])
        self.assertEqual("local-bypass", lines[1])
        self.assertEqual("external-proxy", lines[2])


if __name__ == "__main__":
    unittest.main()
