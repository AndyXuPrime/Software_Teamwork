import sys
import types
from pathlib import Path

import numpy as np


class _DummyFulltextQueryer:
    pass


class _DummyTokenizer:
    @staticmethod
    def tokenize(text):
        return " ".join(str(text).lower().split())

    @staticmethod
    def fine_grained_tokenize(text):
        return text


_fake_query = types.ModuleType("rag.nlp.query")
_fake_query.FulltextQueryer = _DummyFulltextQueryer
_fake_rag_tokenizer = types.ModuleType("rag.nlp.rag_tokenizer")
_fake_rag_tokenizer.tokenize = _DummyTokenizer.tokenize
_fake_rag_tokenizer.fine_grained_tokenize = _DummyTokenizer.fine_grained_tokenize
try:
    import rag.nlp  # noqa: F401
except ModuleNotFoundError:
    _fake_nlp = types.ModuleType("rag.nlp")
    _fake_nlp.__path__ = [str(Path(__file__).resolve().parents[3] / "rag" / "nlp")]
    _fake_nlp.query = _fake_query
    _fake_nlp.rag_tokenizer = _fake_rag_tokenizer
    sys.modules.setdefault("rag.nlp", _fake_nlp)
sys.modules.setdefault("rag.nlp.query", _fake_query)
sys.modules.setdefault("rag.nlp.rag_tokenizer", _fake_rag_tokenizer)
sys.modules.setdefault("common.settings", types.ModuleType("common.settings"))

import rag.nlp.search as search_mod  # noqa: E402
from rag.nlp.search import DEFAULT_SEARCH_FIELDS, Dealer  # noqa: E402
from rag.nlp.retrieval_context import (  # noqa: E402
    assemble_context_packs,
    populate_section_token_fields,
    prepare_rerank_document,
    select_embedding_text,
)


def _dealer():
    return Dealer.__new__(Dealer)


def _search_result(fields):
    ids = list(fields)
    return Dealer.SearchResult(total=len(ids), ids=ids, field=fields)


def _plan(**overrides):
    plan = {
        "normalized_query": "220kV relay table",
        "expanded_queries": ["220kV relay table"],
        "intent": "table_lookup",
        "filters": {"required_tokens": ["220kV"]},
        "weights": {"dense": 0.35, "lexical": 0.25, "section": 0.10, "exact": 0.30},
    }
    plan.update(overrides)
    return plan


def test_default_search_fields_request_optional_section_metadata():
    for field in (
        "section_path",
        "section_title",
        "section_level",
        "block_type",
        "source_block_ids",
        "repair_status",
        "embedding_text",
        "section_title_tks",
        "section_path_tks",
    ):
        assert field in DEFAULT_SEARCH_FIELDS


def test_populate_section_token_fields_from_section_metadata(monkeypatch):
    from rag.nlp import rag_tokenizer

    monkeypatch.setattr(rag_tokenizer, "tokenize", lambda text: " ".join(str(text).lower().split()))
    chunk = {
        "section_title": "Relay Settings",
        "section_path": ["Substation", "Protection"],
        "content_with_weight": "body",
    }

    populate_section_token_fields(chunk)

    assert chunk["section_title_tks"] == "relay settings"
    assert chunk["section_path_tks"] == "substation / protection"


def test_select_embedding_text_prefers_embedding_text_then_questions_then_content():
    assert select_embedding_text(
        {
            "question_kwd": ["Q1", "Q2"],
            "embedding_text": "section context",
            "content_with_weight": "body",
        }
    ) == "section context"
    assert select_embedding_text(
        {
            "question_kwd": ["Q1", "Q2"],
            "content_with_weight": "body",
        }
    ) == "Q1\nQ2"
    assert select_embedding_text({"content_with_weight": "body"}) == "body"


def test_section_and_exact_scores_are_zero_for_legacy_chunks():
    dealer = _dealer()
    sres = _search_result(
        {
            "c1": {
                "content_with_weight": "legacy content",
                "content_ltks": "legacy content",
                "kb_id": "kb_1",
            }
        }
    )

    assert np.allclose(dealer._section_match_scores(_plan(), sres), [0.0])
    assert np.allclose(dealer._exact_match_scores(_plan(), sres), [0.0])
    assert np.allclose(dealer._intent_boost_scores(_plan(), sres), [0.0])


def test_section_exact_and_table_scores_boost_matching_chunks():
    dealer = _dealer()
    sres = _search_result(
        {
            "c1": {
                "content_with_weight": "Relay setting for 220 kV bay",
                "content_ltks": "relay setting 220kv",
                "section_path": "Substation / Relay table",
                "section_title": "220kV Relay Table",
                "block_type": "table",
                "kb_id": "kb_1",
            },
            "c2": {
                "content_with_weight": "Maintenance note",
                "content_ltks": "maintenance note",
                "section_path": "Substation / Notes",
                "section_title": "Maintenance",
                "block_type": "paragraph",
                "kb_id": "kb_1",
            },
        }
    )

    section_scores = dealer._section_match_scores(_plan(), sres)
    exact_scores = dealer._exact_match_scores(_plan(), sres)
    intent_scores = dealer._intent_boost_scores(_plan(), sres)

    assert section_scores[0] > section_scores[1]
    assert exact_scores[0] > exact_scores[1]
    assert intent_scores[0] > intent_scores[1]


def test_repair_rejected_is_penalized_but_not_filtered():
    dealer = _dealer()
    sres = _search_result(
        {
            "c1": {"repair_status": "clean"},
            "c2": {"repair_status": "repair_rejected"},
            "c3": {"repair_status": "repair_skipped"},
        }
    )

    scores = dealer._repair_status_scores(sres)

    assert scores[0] == 0.0
    assert scores[1] < scores[2] < scores[0]


def test_same_section_saturation_moves_overflow_after_diverse_sections():
    dealer = _dealer()
    fields = {
        "c1": {"section_path": "A"},
        "c2": {"section_path": "A"},
        "c3": {"section_path": "A"},
        "c4": {"section_path": "B"},
    }
    sres = _search_result(fields)

    reordered = dealer._apply_same_section_saturation([0, 1, 2, 3], sres, _plan(intent="compare_lookup"), page_size=3)

    assert reordered == [0, 3, 1, 2]


def test_prepare_rerank_document_uses_short_section_context():
    text = prepare_rerank_document(
        {
            "section_path": "Root / Relay",
            "section_title": "220kV table",
            "content_with_weight": "setting value",
            "q_4_vec": [0.1, 0.2],
        },
        "fallback",
    )

    assert text == "Root / Relay / 220kV table\nsetting value"
    assert "0.1" not in text


def test_context_pack_assembles_same_section_adjacent_and_safe_citations():
    chunks = [
        {
            "chunk_id": "c1",
            "doc_id": "doc_1",
            "kb_id": "kb_1",
            "docnm_kwd": "guide.md",
            "content_with_weight": "first",
            "section_path": "Root / Relay",
            "source_block_ids": ["p1-b1"],
            "positions": [[1, 1, 2, 3, 4]],
            "page_num_int": 1,
            "top_int": 1,
            "dense_score": 0.6,
            "lexical_score": 0.2,
        },
        {
            "chunk_id": "c2",
            "doc_id": "doc_1",
            "kb_id": "kb_1",
            "docnm_kwd": "guide.md",
            "content_with_weight": "second",
            "section_path": "Root / Relay",
            "source_block_ids": ["p1-b2"],
            "positions": [[1, 5, 6, 7, 8]],
            "page_num_int": 1,
            "top_int": 2,
            "vector": [0.1, 0.2],
        },
    ]

    packs = assemble_context_packs(chunks, {"context_policy": "chunk"})

    assert packs[0]["primary_chunk"]["content_with_weight"] == "first"
    assert packs[0]["adjacent_chunks"][0]["chunk_id"] == "c2"
    assert packs[0]["citations"] == [
        {
            "kb_id": "kb_1",
            "doc_id": "doc_1",
            "chunk_id": "c1",
            "positions": [[1, 1, 2, 3, 4]],
            "source_block_ids": ["p1-b1"],
        }
    ]
    assert "vector" not in packs[1]["primary_chunk"]


def test_context_pack_orders_adjacent_chunks_from_array_page_and_position_fields():
    chunks = [
        {
            "chunk_id": "c3",
            "doc_id": "doc_1",
            "kb_id": "kb_1",
            "content_with_weight": "third",
            "section_path": "Root / Relay",
            "page_num_int": [1],
            "position_int": [[1, 10, 20, 90, 110]],
        },
        {
            "chunk_id": "c1",
            "doc_id": "doc_1",
            "kb_id": "kb_1",
            "content_with_weight": "first",
            "section_path": "Root / Relay",
            "page_num_int": [1],
            "position_int": [[1, 10, 20, 10, 30]],
        },
        {
            "chunk_id": "c2",
            "doc_id": "doc_1",
            "kb_id": "kb_1",
            "content_with_weight": "second",
            "section_path": "Root / Relay",
            "page_num_int": [1],
            "position_int": [[1, 10, 20, 50, 70]],
        },
    ]

    packs = assemble_context_packs(chunks, {"context_policy": "chunk"})

    assert packs[1]["primary_chunk"]["chunk_id"] == "c1"
    assert [chunk["chunk_id"] for chunk in packs[1]["adjacent_chunks"]] == ["c2"]
    assert [chunk["chunk_id"] for chunk in packs[2]["adjacent_chunks"]] == ["c1", "c3"]


def test_context_pack_keeps_table_chunk_intact_without_adjacent_context():
    packs = assemble_context_packs(
        [
            {
                "chunk_id": "table_1",
                "doc_id": "doc_1",
                "kb_id": "kb_1",
                "content_with_weight": "| a | b |",
                "section_path": "Root / Table",
                "block_type": "table",
            }
        ],
        {"context_policy": "table"},
    )

    assert packs[0]["primary_chunk"]["content_with_weight"] == "| a | b |"
    assert packs[0]["adjacent_chunks"] == []


def test_context_pack_preserves_supported_policy_names():
    chunks = [
        {
            "chunk_id": "c1",
            "doc_id": "doc_1",
            "kb_id": "kb_1",
            "content_with_weight": "first",
            "section_path": "Root / Procedure",
            "page_num_int": 1,
            "top_int": 1,
        },
        {
            "chunk_id": "c2",
            "doc_id": "doc_1",
            "kb_id": "kb_1",
            "content_with_weight": "second",
            "section_path": "Root / Procedure",
            "page_num_int": 1,
            "top_int": 2,
        },
    ]

    for policy in ("chunk", "section", "procedure", "compare", "citation"):
        packs = assemble_context_packs(chunks, {"context_policy": policy})
        assert packs[0]["context_policy"] == policy
        assert packs[0]["adjacent_chunks"][0]["chunk_id"] == "c2"


class _ParentStore:
    def get(self, chunk_id, index_name, kb_ids):
        assert chunk_id == "mom_1"
        assert index_name == "idx"
        assert kb_ids == ["kb_1"]
        return {
            "content_with_weight": "parent content",
            "doc_id": "doc_1",
            "docnm_kwd": "guide.md",
            "kb_id": "kb_1",
            "position_int": [1],
            "doc_type_kwd": "pdf",
            "section_path": "Root / Parent",
            "section_title": "Parent",
            "section_level": 2,
            "block_type": "section",
        }


def test_retrieval_by_children_preserves_parent_and_child_section_metadata(monkeypatch):
    dealer = _dealer()
    dealer.dataStore = _ParentStore()
    monkeypatch.setattr(search_mod, "index_name", lambda scope_id: "idx")
    chunks = [
        {
            "chunk_id": "child_1",
            "mom_id": "mom_1",
            "content_ltks": "child one",
            "content_with_weight": "child one",
            "doc_id": "doc_1",
            "docnm_kwd": "guide.md",
            "kb_id": "kb_1",
            "important_kwd": [],
            "similarity": 0.9,
            "source_block_ids": ["p1-b1"],
        },
        {
            "chunk_id": "child_2",
            "mom_id": "mom_1",
            "content_ltks": "child two",
            "content_with_weight": "child two",
            "doc_id": "doc_1",
            "docnm_kwd": "guide.md",
            "kb_id": "kb_1",
            "important_kwd": [],
            "similarity": 0.7,
            "source_block_ids": ["p1-b2"],
        },
    ]

    merged = dealer.retrieval_by_children(chunks, ["scope_1"])

    assert len(merged) == 1
    assert merged[0]["section_path"] == "Root / Parent"
    assert merged[0]["section_title"] == "Parent"
    assert merged[0]["source_block_ids"] == ["p1-b1", "p1-b2"]
