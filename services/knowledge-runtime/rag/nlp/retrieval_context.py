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
from __future__ import annotations

import json
from dataclasses import asdict, dataclass, field
from typing import Any


def _string(value: Any) -> str:
    if value is None:
        return ""
    if isinstance(value, str):
        return value.strip()
    return str(value).strip()


def _float(value: Any) -> float:
    try:
        return float(value)
    except (TypeError, ValueError):
        return 0.0


def _int_or_none(value: Any) -> int | None:
    try:
        return int(value)
    except (TypeError, ValueError):
        return None


def _first_int_or_none(value: Any) -> int | None:
    if isinstance(value, (list, tuple)):
        for item in value:
            parsed = _first_int_or_none(item)
            if parsed is not None:
                return parsed
        return None
    return _int_or_none(value)


def _top_from_positions(positions: Any) -> int | None:
    if not isinstance(positions, (list, tuple)) or not positions:
        return None
    first = positions[0]
    if not isinstance(first, (list, tuple)):
        return None
    # Existing runtime position tuples are (page, left, right, top, bottom).
    # Keep index 3 first, then accept index 2 for older/misaligned producers.
    for index in (3, 2):
        if len(first) > index:
            parsed = _int_or_none(first[index])
            if parsed is not None:
                return parsed
    return None


def _string_list(value: Any) -> list[str]:
    if value is None or value == "":
        return []
    if isinstance(value, list):
        return [str(item).strip() for item in value if str(item).strip()]
    return [str(value).strip()]


def _text_value(value: Any) -> str:
    if isinstance(value, (list, tuple)):
        return " / ".join(_string_list(list(value)))
    return _string(value)


def _json_safe(value: Any) -> Any:
    try:
        json.dumps(value)
        return value
    except (TypeError, ValueError):
        return str(value)


@dataclass
class ScoreComponents:
    dense: float = 0.0
    lexical: float = 0.0
    section: float = 0.0
    exact: float = 0.0
    base: float = 0.0

    def as_dict(self) -> dict[str, float]:
        return asdict(self)


@dataclass
class Citation:
    kb_id: str
    doc_id: str
    chunk_id: str
    positions: list[Any] = field(default_factory=list)
    source_block_ids: list[str] = field(default_factory=list)

    def as_dict(self) -> dict[str, Any]:
        return {
            "kb_id": self.kb_id,
            "doc_id": self.doc_id,
            "chunk_id": self.chunk_id,
            "positions": _json_safe(self.positions),
            "source_block_ids": self.source_block_ids,
        }


@dataclass
class CandidateChunk:
    chunk_id: str
    doc_id: str
    kb_id: str
    content: str
    document_name: str = ""
    section_path: str = ""
    section_title: str = ""
    section_level: int | None = None
    block_type: str = "unknown"
    source_block_ids: list[str] = field(default_factory=list)
    positions: list[Any] = field(default_factory=list)
    repair_status: str = ""
    chunk_order: int | None = None
    page_number: int | None = None
    top: int | None = None
    score: ScoreComponents = field(default_factory=ScoreComponents)

    @classmethod
    def from_raw(cls, raw: dict[str, Any]) -> "CandidateChunk":
        positions = raw.get("positions") or raw.get("position_int") or []
        top = _first_int_or_none(raw.get("top_int") or raw.get("top"))
        if top is None:
            top = _top_from_positions(positions)
        return cls(
            chunk_id=_string(raw.get("chunk_id") or raw.get("id")),
            doc_id=_string(raw.get("doc_id") or raw.get("document_id")),
            kb_id=_string(raw.get("kb_id") or raw.get("dataset_id")),
            content=_string(raw.get("content_with_weight") or raw.get("content")),
            document_name=_string(raw.get("docnm_kwd") or raw.get("document_name") or raw.get("doc_name")),
            section_path=_string(raw.get("section_path") or raw.get("sectionPath")),
            section_title=_string(raw.get("section_title") or raw.get("sectionTitle")),
            section_level=_first_int_or_none(raw.get("section_level") or raw.get("sectionLevel")),
            block_type=_string(raw.get("block_type") or raw.get("blockType") or "unknown"),
            source_block_ids=_string_list(raw.get("source_block_ids") or raw.get("sourceBlockIds")),
            positions=positions,
            repair_status=_string(raw.get("repair_status") or raw.get("repairStatus")),
            chunk_order=_first_int_or_none(raw.get("chunk_order_int") or raw.get("chunkOrder")),
            page_number=_first_int_or_none(raw.get("page_num_int") or raw.get("pageNumber")),
            top=top,
            score=ScoreComponents(
                dense=_float(raw.get("dense_score") or raw.get("vector_similarity")),
                lexical=_float(raw.get("lexical_score") or raw.get("term_similarity")),
                section=_float(raw.get("section_score")),
                exact=_float(raw.get("exact_score")),
                base=_float(raw.get("base_score") or raw.get("similarity")),
            ),
        )

    def section_key(self) -> str:
        return self.section_path or self.section_title

    def order_key(self) -> tuple[int, int, int, str]:
        return (
            self.page_number if self.page_number is not None else 1 << 30,
            self.top if self.top is not None else 1 << 30,
            self.chunk_order if self.chunk_order is not None else 1 << 30,
            self.chunk_id,
        )

    def citation(self) -> Citation:
        return Citation(
            kb_id=self.kb_id,
            doc_id=self.doc_id,
            chunk_id=self.chunk_id,
            positions=self.positions,
            source_block_ids=self.source_block_ids,
        )

    def safe_summary(self) -> dict[str, Any]:
        out = {
            "chunk_id": self.chunk_id,
            "doc_id": self.doc_id,
            "kb_id": self.kb_id,
            "docnm_kwd": self.document_name,
            "content_with_weight": self.content,
            "section_path": self.section_path,
            "section_title": self.section_title,
            "section_level": self.section_level,
            "block_type": self.block_type,
            "source_block_ids": self.source_block_ids,
            "positions": _json_safe(self.positions),
            "repair_status": self.repair_status,
            "score_components": self.score.as_dict(),
        }
        return {key: value for key, value in out.items() if value not in ("", None, [])}


@dataclass
class ContextPack:
    primary_chunk: dict[str, Any]
    section_path: str = ""
    adjacent_chunks: list[dict[str, Any]] = field(default_factory=list)
    citations: list[dict[str, Any]] = field(default_factory=list)
    context_policy: str = "chunk"

    def as_dict(self) -> dict[str, Any]:
        return {
            "primary_chunk": self.primary_chunk,
            "section_path": self.section_path,
            "adjacent_chunks": self.adjacent_chunks,
            "citations": self.citations,
            "context_policy": self.context_policy,
        }


def prepare_rerank_document(raw: dict[str, Any], fallback_text: str = "") -> str:
    candidate = CandidateChunk.from_raw(raw)
    section = " / ".join(part for part in [candidate.section_path, candidate.section_title] if part)
    content = candidate.content or fallback_text
    if section:
        return f"{section}\n{content}".strip()
    return content.strip() or fallback_text.strip()


def normalize_candidates(chunks: list[dict[str, Any]]) -> list[CandidateChunk]:
    return [CandidateChunk.from_raw(chunk) for chunk in chunks]


def populate_section_token_fields(chunk: dict[str, Any]) -> dict[str, Any]:
    from rag.nlp import rag_tokenizer

    for source_field, token_field in (
        ("section_title", "section_title_tks"),
        ("section_path", "section_path_tks"),
    ):
        text = _text_value(chunk.get(source_field))
        if text:
            chunk[token_field] = rag_tokenizer.tokenize(text)
    return chunk


def select_embedding_text(chunk: dict[str, Any], use_question_kwd: bool = True) -> str:
    embedding_text = _text_value(chunk.get("embedding_text"))
    if embedding_text:
        return embedding_text
    if use_question_kwd:
        question_kwd = chunk.get("question_kwd") or []
        if isinstance(question_kwd, str):
            question_kwd = [question_kwd]
        questions = [str(question).strip() for question in question_kwd if str(question).strip()]
        if questions:
            return "\n".join(questions)
    return _text_value(chunk.get("content_with_weight"))


def _context_policy(plan: dict[str, Any] | None) -> str:
    if not isinstance(plan, dict):
        return "chunk"
    policy = str(plan.get("context_policy") or "chunk")
    return policy if policy in {"chunk", "section", "table", "procedure", "compare", "citation"} else "chunk"


def assemble_context_packs(chunks: list[dict[str, Any]], plan: dict[str, Any] | None = None) -> list[dict[str, Any]]:
    candidates = normalize_candidates(chunks)
    by_section: dict[str, list[CandidateChunk]] = {}
    for candidate in candidates:
        key = candidate.section_key()
        if key:
            by_section.setdefault(key, []).append(candidate)
    for group in by_section.values():
        group.sort(key=lambda candidate: candidate.order_key())

    policy = _context_policy(plan)
    packs: list[dict[str, Any]] = []
    for candidate in candidates:
        adjacent: list[dict[str, Any]] = []
        if policy != "table" and candidate.section_key():
            peers = by_section.get(candidate.section_key(), [])
            peer_index = next((idx for idx, peer in enumerate(peers) if peer.chunk_id == candidate.chunk_id), -1)
            if peer_index >= 0:
                for peer in peers[max(0, peer_index - 1): peer_index] + peers[peer_index + 1: peer_index + 2]:
                    adjacent.append(peer.safe_summary())

        pack = ContextPack(
            primary_chunk=candidate.safe_summary(),
            section_path=candidate.section_path,
            adjacent_chunks=adjacent,
            citations=[candidate.citation().as_dict()],
            context_policy=policy,
        )
        packs.append(pack.as_dict())
    return packs
