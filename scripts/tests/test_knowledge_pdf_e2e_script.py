import contextlib
import importlib.util
import io
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock


SCRIPT_PATH = Path(__file__).resolve().parents[1] / "local" / "knowledge-pdf-e2e.py"


def load_script_module():
    module_name = "knowledge_pdf_e2e"
    spec = importlib.util.spec_from_file_location(module_name, SCRIPT_PATH)
    if spec is None or spec.loader is None:
        raise AssertionError(f"failed to load {SCRIPT_PATH}")
    module = importlib.util.module_from_spec(spec)
    sys.modules[module_name] = module
    spec.loader.exec_module(module)
    return module


class KnowledgePdfE2EScriptTests(unittest.TestCase):
    def test_pdf_argument_is_required(self) -> None:
        module = load_script_module()

        with mock.patch.object(sys, "argv", ["knowledge-pdf-e2e.py"]):
            with contextlib.redirect_stderr(io.StringIO()):
                with self.assertRaises(SystemExit) as raised:
                    module.parse_args()

        self.assertEqual(2, raised.exception.code)

    def test_missing_pdf_fails_before_loading_runtime_config(self) -> None:
        module = load_script_module()

        with tempfile.TemporaryDirectory() as directory:
            missing_pdf = Path(directory) / "missing.pdf"
            with mock.patch.object(sys, "argv", ["knowledge-pdf-e2e.py", str(missing_pdf)]):
                with mock.patch.object(module, "load_config", side_effect=AssertionError("config should not load")):
                    with self.assertRaises(SystemExit) as raised:
                        module.run()

        message = str(raised.exception)
        self.assertIn("PDF fixture not found before starting Knowledge smoke", message)
        self.assertIn("Pass an existing local PDF path", message)

    def test_existing_pdf_path_resolves(self) -> None:
        module = load_script_module()

        with tempfile.TemporaryDirectory() as directory:
            pdf = Path(directory) / "sample.pdf"
            pdf.write_bytes(b"%PDF-1.4\n%%EOF\n")

            self.assertEqual(pdf.resolve(), module.resolve_pdf_path(str(pdf)))


if __name__ == "__main__":
    unittest.main()
