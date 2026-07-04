from api.apps.restful_apis import chunk_api


def test_chunk_embedding_provider_returns_document_embedding_model(monkeypatch):
    monkeypatch.setattr(chunk_api.DocumentService, "get_embd_id", lambda document_id: "bge@default@AI_GATEWAY")

    assert chunk_api._chunk_embedding_provider("doc_1") == "bge@default@AI_GATEWAY"


def test_list_chunk_fields_avoid_backend_missing_columns():
    fields = chunk_api._list_chunk_fields()

    assert "token_count" not in fields
    assert "token_num" not in fields
    assert "embedding_dimension_int" not in fields


def test_chunk_from_search_result_does_not_read_doc_store(monkeypatch):
    class FailingDocStore:
        def get(self, *args, **kwargs):
            raise AssertionError("list chunk mapping must not fetch full chunk vectors")

    monkeypatch.setattr(chunk_api.settings, "docStoreConn", FailingDocStore())

    result = chunk_api._chunk_from_search_result(
        "chunk_1",
        {
            "content_with_weight": "alpha beta",
            "doc_id": "doc_1",
            "docnm_kwd": "doc.pdf",
            "kb_id": "kb_1",
            "available_int": "1",
        },
        {},
        "",
        "bge@default@AI_GATEWAY",
    )

    assert result["embedding_provider"] == "bge@default@AI_GATEWAY"
    assert "embedding_dimension" not in result
    assert "token_count" not in result
