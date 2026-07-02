"""Lightweight CI smoke for the vendored deepdoc PDF parser.

The full upstream deepdoc parser tests load heavyweight parser dependencies and
the unit-test conftest downloads NLTK data. This smoke keeps CI deterministic
while proving the parser module itself still imports under mocked optional
dependencies and that the garbled-text helper remains callable.
"""

from __future__ import annotations

import importlib.util
import sys
from pathlib import Path
from unittest import mock


MOCK_MODULES = [
    "numpy",
    "np",
    "pdfplumber",
    "xgboost",
    "xgb",
    "huggingface_hub",
    "PIL",
    "PIL.Image",
    "pypdf",
    "sklearn",
    "sklearn.cluster",
    "sklearn.metrics",
    "common",
    "common.constants",
    "common.file_utils",
    "common.misc_utils",
    "common.settings",
    "common.token_utils",
    "deepdoc",
    "deepdoc.vision",
    "deepdoc.parser",
    "deepdoc.parser.utils",
    "rag",
    "rag.nlp",
    "rag.prompts",
    "rag.prompts.generator",
]


def test_deepdoc_pdf_parser_garbled_helpers_import_under_lightweight_mocks(monkeypatch):
    for module_name in MOCK_MODULES:
        monkeypatch.setitem(sys.modules, module_name, mock.MagicMock())

    module_path = Path(__file__).resolve().parents[2] / "deepdoc" / "parser" / "pdf_parser.py"
    spec = importlib.util.spec_from_file_location("pdf_parser_ci_smoke", module_path)
    assert spec is not None
    assert spec.loader is not None

    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)

    parser = module.RAGFlowPdfParser
    assert parser._is_garbled_char("\uE000") is True
    assert parser._is_garbled_text("(cid:123)") is True
    assert parser._is_garbled_text("normal text") is False
