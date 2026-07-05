import json
import os
import subprocess
import textwrap
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]
WORKFLOW_PATH = REPO_ROOT / ".github" / "workflows" / "docker-deploy-checks.yml"


class DockerDeployWorkflowDetectTests(unittest.TestCase):
    def test_cloud_dockerfile_changes_trigger_policy_and_cloud_compose(self) -> None:
        outputs = self.run_detect(["deploy/docker/full/go-service.Dockerfile"])

        self.assertEqual("true", outputs["has-policy-check"])
        self.assertEqual("true", outputs["has-compose-files"])
        self.assertEqual(["deploy/docker-compose.cloud.yml"], json.loads(outputs["compose-files"]))

    def test_cloud_docker_support_changes_trigger_policy_and_cloud_compose(self) -> None:
        outputs = self.run_detect(["deploy/docker/full/nginx.conf"])

        self.assertEqual("true", outputs["has-policy-check"])
        self.assertEqual(["deploy/docker-compose.cloud.yml"], json.loads(outputs["compose-files"]))

    def test_unrelated_docs_do_not_trigger_docker_policy(self) -> None:
        outputs = self.run_detect(["docs/services/gateway/README.md"])

        self.assertEqual("false", outputs["has-policy-check"])
        self.assertEqual("false", outputs["has-compose-files"])
        self.assertEqual([], json.loads(outputs["compose-files"]))

    def run_detect(self, changed_files: list[str]) -> dict[str, str]:
        script = extract_github_script(WORKFLOW_PATH)
        wrapper = textwrap.dedent(
            f"""
            const changedFiles = JSON.parse(process.env.CHANGED_FILES);
            const outputs = {{}};
            const core = {{
              info: () => {{}},
              setOutput: (name, value) => {{
                outputs[name] = String(value);
              }},
            }};
            const context = {{
              eventName: 'pull_request',
              repo: {{ owner: 'owner', repo: 'repo' }},
              payload: {{ pull_request: {{ number: 1 }} }},
            }};
            const github = {{
              paginate: async () => changedFiles.map((filename) => ({{ filename }})),
              rest: {{
                pulls: {{ listFiles: async () => {{}} }},
                repos: {{
                  compareCommitsWithBasehead: async () => ({{
                    data: {{ files: changedFiles.map((filename) => ({{ filename }})) }},
                  }}),
                }},
              }},
            }};
            (async () => {{
            {indent(script, "              ")}
              process.stdout.write(JSON.stringify(outputs));
            }})().catch((error) => {{
              console.error(error);
              process.exit(1);
            }});
            """
        )
        result = subprocess.run(
            ["node", "-e", wrapper],
            cwd=REPO_ROOT,
            env={**os.environ, "CHANGED_FILES": json.dumps(changed_files)},
            text=True,
            capture_output=True,
            check=False,
        )
        self.assertEqual(0, result.returncode, result.stderr)
        return json.loads(result.stdout)


def extract_github_script(path: Path) -> str:
    lines = path.read_text(encoding="utf-8").splitlines()
    for index, line in enumerate(lines):
        if line.strip() != "script: |":
            continue
        collected: list[str] = []
        for script_line in lines[index + 1 :]:
            if script_line and not script_line.startswith("            "):
                break
            collected.append(script_line[12:] if script_line.startswith("            ") else "")
        return "\n".join(collected)
    raise AssertionError("actions/github-script block not found")


def indent(content: str, prefix: str) -> str:
    return "\n".join(prefix + line if line else "" for line in content.splitlines())


if __name__ == "__main__":
    unittest.main()
