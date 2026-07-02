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
