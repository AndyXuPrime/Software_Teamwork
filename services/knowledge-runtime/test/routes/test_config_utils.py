from pathlib import Path

from common.config_utils import sanitize_for_logging


def test_sanitize_for_logging_masks_nested_model_api_keys():
    value = {
        "user_default_llm": {
            "default_models": {
                "embedding_model": {
                    "factory": "SILICONFLOW",
                    "api_key": "sk-secret",
                    "base_url": "https://api.example/v1",
                }
            }
        },
        "token": "session-token",
    }

    sanitized = sanitize_for_logging(value)

    assert sanitized["user_default_llm"]["default_models"]["embedding_model"]["api_key"] == "********"
    assert sanitized["user_default_llm"]["default_models"]["embedding_model"]["base_url"] == "https://api.example/v1"
    assert sanitized["token"] == "********"


def test_runtime_database_type_defaults_to_postgres():
    settings_source = Path(__file__).parents[2] / "common" / "settings.py"
    content = settings_source.read_text(encoding="utf-8")

    assert 'DEFAULT_DATABASE_TYPE = "postgres"' in content
    assert 'os.getenv("DB_TYPE", DEFAULT_DATABASE_TYPE)' in content
