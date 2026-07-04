#
#  Copyright 2024 The InfiniFlow Authors. All Rights Reserved.
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.
#
import json
import re
from dataclasses import asdict, dataclass, field
from typing import Any, Protocol


INTENT_FACT = "fact_lookup"
INTENT_SECTION = "section_lookup"
INTENT_TABLE = "table_lookup"
INTENT_NUMERIC = "numeric_lookup"
INTENT_PROCEDURE = "procedure_lookup"
INTENT_COMPARE = "compare_lookup"
INTENT_CITATION = "citation_lookup"

ALLOWED_INTENTS = {
    INTENT_FACT,
    INTENT_SECTION,
    INTENT_TABLE,
    INTENT_NUMERIC,
    INTENT_PROCEDURE,
    INTENT_COMPARE,
    INTENT_CITATION,
}

INTENT_WEIGHTS = {
    INTENT_FACT: {"dense": 0.65, "lexical": 0.25, "section": 0.10, "exact": 0.00},
    INTENT_SECTION: {"dense": 0.45, "lexical": 0.20, "section": 0.35, "exact": 0.00},
    INTENT_TABLE: {"dense": 0.35, "lexical": 0.25, "section": 0.10, "exact": 0.30},
    INTENT_NUMERIC: {"dense": 0.30, "lexical": 0.25, "section": 0.10, "exact": 0.35},
    INTENT_PROCEDURE: {"dense": 0.50, "lexical": 0.25, "section": 0.25, "exact": 0.00},
    INTENT_COMPARE: {"dense": 0.55, "lexical": 0.25, "section": 0.20, "exact": 0.00},
    INTENT_CITATION: {"dense": 0.45, "lexical": 0.25, "section": 0.30, "exact": 0.00},
}

CONTEXT_POLICY_BY_INTENT = {
    INTENT_FACT: "chunk",
    INTENT_SECTION: "section",
    INTENT_TABLE: "table",
    INTENT_NUMERIC: "chunk",
    INTENT_PROCEDURE: "procedure",
    INTENT_COMPARE: "compare",
    INTENT_CITATION: "citation",
}

BASE_TARGET_FIELDS = ["content", "title", "question"]
SECTION_TARGET_FIELDS = ["section_title", "section_path"]
EXACT_TOKEN_RE = re.compile(
    r"""
    (?:
        \b[A-Z]{2,}[-_]?\d+[A-Z0-9._/-]*\b
        |
        \b\d+(?:\.\d+)?\s*(?:kV|KV|V|A|MW|MVA|Hz|kg|mm|cm|m|%|percent|year|day|h|ms|s)\b
        |
        \b\d{4}[-/.]\d{1,2}[-/.]\d{1,2}\b
        |
        \b\d+(?:\.\d+)?\b
    )
    """,
    re.VERBOSE,
)


@dataclass
class RetrievalPlan:
    original_query: str
    normalized_query: str
    expanded_queries: list[str]
    intent: str
    target_fields: list[str]
    filters: dict[str, Any] = field(default_factory=dict)
    weights: dict[str, float] = field(default_factory=dict)
    needs_rerank: bool = True
    context_policy: str = "chunk"

    def as_dict(self) -> dict[str, Any]:
        return asdict(self)


class LLMQueryPlanner(Protocol):
    def plan(self, query: str, metadata: dict[str, Any]) -> str | dict[str, Any]:
        ...


class AsyncLLMQueryPlanner(Protocol):
    async def plan(self, query: str, metadata: dict[str, Any]) -> str | dict[str, Any]:
        ...


class RuntimeLLMQueryPlanner:
    def __init__(self, chat_mdl, timeout_seconds: float = 3.0):
        self.chat_mdl = chat_mdl
        self.timeout_seconds = timeout_seconds

    async def plan(self, query: str, metadata: dict[str, Any]) -> str:
        import asyncio

        safe_query = normalize_query(query)[:1024]
        safe_metadata = {
            "available_intents": sorted(ALLOWED_INTENTS),
            "safe_metadata": {
                key: value
                for key, value in (metadata or {}).items()
                if key in {"dataset_count", "has_metadata_filter", "has_doc_filter"}
            },
        }
        system = (
            "You create retrieval query plans only. Return strict JSON. "
            "Do not answer the user's question. Do not include answer, response, "
            "final_answer, prompt, vectors, provider payloads, credentials, or raw documents."
        )
        user = json.dumps(
            {
                "query": safe_query,
                "metadata": safe_metadata,
                "schema": {
                    "normalized_query": "string",
                    "expanded_queries": ["string"],
                    "intent": sorted(ALLOWED_INTENTS),
                    "target_fields": ["content", "section_title", "section_path"],
                    "filters": {"required_tokens": ["string"], "block_types": ["string"]},
                    "weights": {"dense": 0.0, "lexical": 0.0, "section": 0.0, "exact": 0.0},
                    "needs_rerank": True,
                    "context_policy": ["chunk", "section", "table", "procedure", "compare", "citation"],
                },
            },
            ensure_ascii=False,
        )
        return await asyncio.wait_for(
            self.chat_mdl.async_chat(system, [{"role": "user", "content": user}], {"temperature": 0.0}),
            timeout=self.timeout_seconds,
        )


def normalize_query(query: str) -> str:
    return re.sub(r"\s+", " ", (query or "").strip())


def extract_required_tokens(query: str) -> list[str]:
    tokens = []
    seen = set()
    for match in EXACT_TOKEN_RE.finditer(query or ""):
        token = re.sub(r"\s+", "", match.group(0))
        key = token.lower()
        if key and key not in seen:
            seen.add(key)
            tokens.append(token)
    return tokens[:16]


def detect_intent(query: str) -> str:
    q = normalize_query(query).lower()
    if not q:
        return INTENT_FACT

    if re.search(r"\b(table|spreadsheet|row|column|cell|matrix)\b|表格|清单|列表|行列|公式", q):
        return INTENT_TABLE
    if re.search(r"\b(compare|contrast|difference|versus|vs\.?)\b|对比|比较|差异|区别", q):
        return INTENT_COMPARE
    if re.search(r"\b(step|procedure|process|workflow|how to|checklist)\b|步骤|流程|方法|如何|怎么|规程", q):
        return INTENT_PROCEDURE
    if re.search(r"\b(cite|citation|source|page|reference|origin)\b|引用|出处|来源|页码|原文", q):
        return INTENT_CITATION
    if re.search(r"\b(section|chapter|heading|clause|part)\b|章节|小节|标题|条款|目录", q):
        return INTENT_SECTION
    if extract_required_tokens(q) or re.search(r"\b(number|value|date|voltage|current|capacity|rate)\b|数值|日期|电压|电流|容量|比例", q):
        return INTENT_NUMERIC
    return INTENT_FACT


def expanded_queries(normalized_query: str) -> list[str]:
    expansions = [normalized_query] if normalized_query else []
    alias_map = {
        "voltage": "电压",
        "current": "电流",
        "capacity": "容量",
        "transformer": "变压器",
        "maintenance": "维护",
        "procedure": "流程",
        "table": "表格",
        "section": "章节",
    }
    lowered = normalized_query.lower()
    for source, alias in alias_map.items():
        if source in lowered and alias not in expansions:
            expansions.append(alias)
        if alias in normalized_query and source not in expansions:
            expansions.append(source)
    return expansions[:6]


def target_fields(intent: str) -> list[str]:
    fields = list(BASE_TARGET_FIELDS)
    if intent in {INTENT_SECTION, INTENT_PROCEDURE, INTENT_CITATION, INTENT_COMPARE}:
        fields.extend(SECTION_TARGET_FIELDS)
    if intent == INTENT_TABLE:
        fields.extend(SECTION_TARGET_FIELDS)
        fields.append("block_type")
    if intent == INTENT_NUMERIC:
        fields.append("numbers")
    return list(dict.fromkeys(fields))


def deterministic_plan(query: str) -> RetrievalPlan:
    original = query or ""
    normalized = normalize_query(original)
    intent = detect_intent(normalized)
    required_tokens = extract_required_tokens(normalized)
    filters: dict[str, Any] = {}
    if intent == INTENT_TABLE:
        filters["block_types"] = ["table"]
    if required_tokens:
        filters["required_tokens"] = required_tokens
    return RetrievalPlan(
        original_query=original,
        normalized_query=normalized,
        expanded_queries=expanded_queries(normalized),
        intent=intent,
        target_fields=target_fields(intent),
        filters=filters,
        weights=INTENT_WEIGHTS[intent].copy(),
        needs_rerank=intent in {INTENT_SECTION, INTENT_TABLE, INTENT_NUMERIC, INTENT_COMPARE, INTENT_CITATION},
        context_policy=CONTEXT_POLICY_BY_INTENT[intent],
    )


def _coerce_llm_payload(payload: str | dict[str, Any]) -> dict[str, Any]:
    if isinstance(payload, str):
        text = payload.strip()
        if text.startswith("```"):
            text = re.sub(r"^```(?:json)?\s*", "", text, flags=re.IGNORECASE)
            text = re.sub(r"\s*```$", "", text)
        payload = json.loads(text)
    if not isinstance(payload, dict):
        raise ValueError("planner output must be a JSON object")
    forbidden_keys = {"answer", "response", "final_answer", "content"}
    if forbidden_keys.intersection(payload):
        raise ValueError("planner output must not include answer content")
    return payload


def _validated_plan(payload: str | dict[str, Any], fallback: RetrievalPlan) -> RetrievalPlan:
    data = _coerce_llm_payload(payload)
    intent = str(data.get("intent") or fallback.intent)
    if intent not in ALLOWED_INTENTS:
        raise ValueError("planner output has an unsupported intent")

    normalized = normalize_query(str(data.get("normalized_query") or fallback.normalized_query))
    expanded = data.get("expanded_queries", fallback.expanded_queries)
    if not isinstance(expanded, list) or any(not isinstance(item, str) for item in expanded):
        raise ValueError("expanded_queries must be a string list")
    expanded = [normalize_query(item) for item in expanded if normalize_query(item)][:8]
    if not expanded and normalized:
        expanded = [normalized]

    raw_fields = data.get("target_fields", fallback.target_fields)
    if not isinstance(raw_fields, list) or any(not isinstance(item, str) for item in raw_fields):
        raise ValueError("target_fields must be a string list")
    fields = list(dict.fromkeys([item for item in raw_fields if item]))

    filters = data.get("filters", fallback.filters)
    if not isinstance(filters, dict):
        raise ValueError("filters must be an object")

    weights = data.get("weights", INTENT_WEIGHTS[intent])
    if not isinstance(weights, dict):
        raise ValueError("weights must be an object")
    safe_weights = INTENT_WEIGHTS[intent].copy()
    for key in ("dense", "lexical", "section", "exact"):
        if key in weights:
            value = float(weights[key])
            if value < 0 or value > 1:
                raise ValueError("weights must stay within [0, 1]")
            safe_weights[key] = value

    context_policy = str(data.get("context_policy") or CONTEXT_POLICY_BY_INTENT[intent])
    if context_policy not in {"chunk", "section", "table", "procedure", "compare", "citation"}:
        raise ValueError("unsupported context policy")

    return RetrievalPlan(
        original_query=fallback.original_query,
        normalized_query=normalized,
        expanded_queries=expanded,
        intent=intent,
        target_fields=fields,
        filters=filters,
        weights=safe_weights,
        needs_rerank=bool(data.get("needs_rerank", fallback.needs_rerank)),
        context_policy=context_policy,
    )


def plan_query(
    query: str,
    llm_planner: LLMQueryPlanner | None = None,
    use_llm: bool = False,
    metadata: dict[str, Any] | None = None,
) -> RetrievalPlan:
    fallback = deterministic_plan(query)
    if not use_llm or llm_planner is None:
        return fallback
    try:
        return _validated_plan(llm_planner.plan(fallback.normalized_query, metadata or {}), fallback)
    except Exception:
        return fallback


async def async_plan_query(
    query: str,
    llm_planner: AsyncLLMQueryPlanner | None = None,
    use_llm: bool = False,
    metadata: dict[str, Any] | None = None,
) -> RetrievalPlan:
    fallback = deterministic_plan(query)
    if not use_llm or llm_planner is None:
        return fallback
    try:
        return _validated_plan(await llm_planner.plan(fallback.normalized_query, metadata or {}), fallback)
    except Exception:
        return fallback
