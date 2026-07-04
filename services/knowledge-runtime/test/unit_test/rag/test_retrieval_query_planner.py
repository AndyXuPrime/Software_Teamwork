import json
import sys
import types
from pathlib import Path

import pytest


try:
    import rag.nlp  # noqa: F401
except ModuleNotFoundError:
    _fake_nlp = types.ModuleType("rag.nlp")
    _fake_nlp.__path__ = [str(Path(__file__).resolve().parents[3] / "rag" / "nlp")]
    sys.modules.setdefault("rag.nlp", _fake_nlp)

from rag.nlp.retrieval_planner import (
    INTENT_CITATION,
    INTENT_COMPARE,
    INTENT_FACT,
    INTENT_NUMERIC,
    INTENT_PROCEDURE,
    INTENT_SECTION,
    INTENT_TABLE,
    async_plan_query,
    deterministic_plan,
    plan_query,
)


def test_deterministic_planner_detects_supported_intents():
    cases = [
        ("What is the rated voltage of the transformer?", INTENT_NUMERIC, "chunk"),
        ("Find the section about relay calibration", INTENT_SECTION, "section"),
        ("Show the table of inspection results", INTENT_TABLE, "table"),
        ("How to perform the shutdown procedure?", INTENT_PROCEDURE, "procedure"),
        ("Compare breaker A versus breaker B", INTENT_COMPARE, "compare"),
        ("Give the citation and page source", INTENT_CITATION, "citation"),
        ("What is transformer oil?", INTENT_FACT, "chunk"),
    ]

    for query, intent, context_policy in cases:
        plan = deterministic_plan(query)
        assert plan.intent == intent
        assert plan.context_policy == context_policy
        assert plan.normalized_query
        assert plan.expanded_queries
        assert "content" in plan.target_fields
        assert set(plan.weights) == {"dense", "lexical", "section", "exact"}


def test_deterministic_planner_extracts_required_numeric_tokens():
    plan = deterministic_plan("Find 220 kV relay setting on 2024-05-01")

    assert plan.intent == INTENT_NUMERIC
    assert "required_tokens" in plan.filters
    assert "2024-05-01" in plan.filters["required_tokens"]
    assert any(token.lower() == "220kv" for token in plan.filters["required_tokens"])


class _Planner:
    def __init__(self, payload):
        self.payload = payload

    def plan(self, query, metadata):
        assert query == "relay table"
        assert metadata == {"safe": True}
        return self.payload


def test_llm_planner_accepts_strict_json_plan():
    payload = json.dumps(
        {
            "normalized_query": "relay table",
            "expanded_queries": ["relay table", "继电器 表格"],
            "intent": INTENT_TABLE,
            "target_fields": ["content", "section_title", "block_type"],
            "filters": {"block_types": ["table"]},
            "weights": {"dense": 0.3, "lexical": 0.2, "section": 0.2, "exact": 0.3},
            "needs_rerank": True,
            "context_policy": "table",
        }
    )

    plan = plan_query("relay table", llm_planner=_Planner(payload), use_llm=True, metadata={"safe": True})

    assert plan.intent == INTENT_TABLE
    assert plan.context_policy == "table"
    assert plan.filters == {"block_types": ["table"]}


def test_llm_planner_falls_back_on_answer_content_or_invalid_json():
    answer_payload = {"intent": INTENT_TABLE, "answer": "Use row 3"}
    invalid_payload = "not json"

    assert plan_query("relay table", llm_planner=_Planner(answer_payload), use_llm=True).intent == INTENT_TABLE
    assert plan_query("What is transformer oil?", llm_planner=_Planner(invalid_payload), use_llm=True).intent == INTENT_FACT


class _AsyncPlanner:
    def __init__(self, payload):
        self.payload = payload

    async def plan(self, query, metadata):
        return self.payload


@pytest.mark.asyncio
async def test_async_llm_planner_falls_back_on_invalid_output():
    plan = await async_plan_query(
        "relay table",
        llm_planner=_AsyncPlanner({"intent": INTENT_TABLE, "content": "answer leak"}),
        use_llm=True,
    )

    assert plan.intent == INTENT_TABLE
    assert plan.context_policy == "table"
