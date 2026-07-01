import importlib


def test_chunk_feedback_service_imports():
    module = importlib.import_module("api.db.services.chunk_feedback_service")
    assert hasattr(module, "ChunkFeedbackService")
