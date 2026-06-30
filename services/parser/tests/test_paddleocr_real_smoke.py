import base64
import importlib.util
import os
from pathlib import Path
from typing import NoReturn

import pytest

from parser_service.backends.paddleocr import PaddleOCRBackend
from parser_service.service import ParseRequest

pytestmark = pytest.mark.paddleocr_smoke


FIXTURE_B64 = Path(__file__).parent / "fixtures" / "paddleocr_smoke.png.b64"


def test_real_paddleocr_model_parses_fixture_text(tmp_path):
    if os.environ.get("PARSER_PADDLEOCR_SMOKE", "").strip() != "1":
        pytest.skip("set PARSER_PADDLEOCR_SMOKE=1 to run the real PaddleOCR model smoke")

    _require_paddleocr_runtime()
    _require_model_policy()

    image_bytes = base64.b64decode(FIXTURE_B64.read_text(encoding="ascii"))
    image_path = tmp_path / "paddleocr-smoke.png"
    image_path.write_bytes(image_bytes)

    backend = PaddleOCRBackend(
        lang=_env("PARSER_PADDLEOCR_LANG", "PADDLEOCR_LANG", default="ch"),
        device=_env("PARSER_PADDLEOCR_DEVICE", "PADDLEOCR_DEVICE", default="cpu"),
        engine=_env("PARSER_PADDLEOCR_ENGINE", "PADDLEOCR_ENGINE", default=""),
        paddlex_config=_env(
            "PARSER_PADDLEOCR_CONFIG_PATH",
            "PADDLEOCR_CONFIG_PATH",
            default="",
        ),
        use_doc_orientation_classify=_bool_env(
            "PARSER_PADDLEOCR_USE_DOC_ORIENTATION_CLASSIFY",
            "PADDLEOCR_USE_DOC_ORIENTATION_CLASSIFY",
        ),
        use_doc_unwarping=_bool_env(
            "PARSER_PADDLEOCR_USE_DOC_UNWARPING",
            "PADDLEOCR_USE_DOC_UNWARPING",
        ),
        use_textline_orientation=_bool_env(
            "PARSER_PADDLEOCR_USE_TEXTLINE_ORIENTATION",
            "PADDLEOCR_USE_TEXTLINE_ORIENTATION",
        ),
    )

    request = ParseRequest(
        document_name=image_path.name,
        content_type="image/png",
        size_bytes=len(image_bytes),
        data=image_bytes,
    )

    try:
        parsed = backend.parse(request)
    except Exception as exc:
        _fail(
            "real PaddleOCR smoke failed while loading the model or parsing the fixture. "
            "Check PaddleOCR/PaddlePaddle installation, model config paths, CPU/GPU device "
            "selection, and whether downloads are allowed for this local run.",
            exc,
        )

    assert parsed.content.strip(), (
        "real PaddleOCR smoke completed but returned empty text; verify the fixture is readable, "
        "the language/device config matches the local model, and model files are complete"
    )


def _require_paddleocr_runtime() -> None:
    missing = [
        module
        for module in ("paddleocr", "paddle")
        if importlib.util.find_spec(module) is None
    ]
    if missing:
        pytest.fail(
            "PARSER_PADDLEOCR_SMOKE=1 requires real OCR dependencies. "
            f"Missing modules: {', '.join(missing)}. "
            "Run `uv sync --group dev --extra paddleocr` from services/parser.",
            pytrace=False,
        )


def _require_model_policy() -> None:
    config_path = _env("PARSER_PADDLEOCR_CONFIG_PATH", "PADDLEOCR_CONFIG_PATH", default="")
    if config_path:
        if not Path(config_path).is_file():
            pytest.fail(
                "PADDLEOCR_CONFIG_PATH/PARSER_PADDLEOCR_CONFIG_PATH must point to an existing "
                "PaddleX config file when real smoke uses local model paths.",
                pytrace=False,
            )
        return

    allow_download = _bool_env("PARSER_PADDLEOCR_ALLOW_DOWNLOAD")
    if not allow_download:
        pytest.fail(
            "PARSER_PADDLEOCR_SMOKE=1 requires either "
            "PADDLEOCR_CONFIG_PATH/PARSER_PADDLEOCR_CONFIG_PATH for prepared local model paths "
            "or PARSER_PADDLEOCR_ALLOW_DOWNLOAD=1 to permit PaddleOCR to use its default model "
            "download/cache behavior.",
            pytrace=False,
        )


def _env(*names: str, default: str) -> str:
    for name in names:
        value = os.environ.get(name, "").strip()
        if value:
            return value
    return default


def _bool_env(*names: str) -> bool:
    value = _env(*names, default="").lower()
    if value in {"", "0", "false", "no", "off"}:
        return False
    if value in {"1", "true", "yes", "on"}:
        return True
    pytest.fail(f"{'/'.join(names)} must be a boolean value", pytrace=False)


def _fail(message: str, exc: Exception) -> NoReturn:
    pytest.fail(f"{message} Original error type: {type(exc).__name__}.", pytrace=False)
