from parser_service.config import Settings


def test_internal_service_token_is_parser_token_fallback(monkeypatch):
    monkeypatch.delenv("PARSER_SERVICE_TOKEN", raising=False)
    monkeypatch.setenv("INTERNAL_SERVICE_TOKEN", "shared-token")

    settings = Settings.from_env()

    assert settings.service_token == "shared-token"
