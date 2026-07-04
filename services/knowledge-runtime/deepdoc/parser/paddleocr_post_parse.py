#  Copyright 2026 The InfiniFlow Authors. All Rights Reserved.
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
import asyncio
import json
import logging
import re
from dataclasses import dataclass, field, replace
from difflib import SequenceMatcher
from typing import Any, Iterable, Protocol

from bs4 import BeautifulSoup

from deepdoc.parser.paddleocr_adapter import BBox, EMPTY_BBOX, PaddleOCRBlock, PaddleOCRPage
from deepdoc.parser.paddleocr_normalizer import PaddleOCRLayoutNormalizer, SemanticSection


_FACT_TOKEN_RE = re.compile(
    r"""
    (?:
        \b\d{1,4}[/-]\d{1,2}(?:[/-]\d{1,4})?\b
        |
        \b\d+(?:\.\d+)?\s*(?:%|[A-Za-z]{1,8}|kV|KV|kg|mm|cm|m|℃|°C|千克|毫米|厘米|米)?\b
        |
        \b[A-Z]{2,}[-_/A-Z0-9]{1,}\b
        |
        \b[A-Za-z]+-\d+\b
    )
    """,
    re.VERBOSE,
)
_MARKDOWN_HEADING_RE = re.compile(r"^(#{1,6})\s+(.+)$")
_NUMBERED_HEADING_RE = re.compile(r"^\s*(\d+(?:\.\d+){0,5})[\s.、:：-]+")
_CONTROL_CHAR_RE = re.compile(r"[\x00-\x08\x0b\x0c\x0e-\x1f]")


@dataclass
class ValidatedSection:
    id: str
    text: str
    block_type: str
    page_start: int
    page_end: int
    bbox: BBox = EMPTY_BBOX
    order: int = 0
    title: str = ""
    level: int = 0
    section_path: list[str] = field(default_factory=list)
    positions: list[tuple[int, float, float, float, float]] = field(default_factory=list)
    source_block_ids: list[str] = field(default_factory=list)
    repair_status: str = "clean"
    quality_flags: list[str] = field(default_factory=list)
    metadata: dict[str, Any] = field(default_factory=dict)

    @classmethod
    def from_semantic(cls, section: SemanticSection, index: int) -> "ValidatedSection":
        metadata = dict(section.metadata or {})
        canonical_id = str(metadata.get("canonical_block_id") or metadata.get("id") or f"sec-{index:04d}")
        source_ids = metadata.get("source_block_ids")
        if isinstance(source_ids, list):
            source_ids = [str(v) for v in source_ids if v]
        if not isinstance(source_ids, list) or not source_ids:
            source_ids = [
                str(v)
                for v in (
                    metadata.get("canonical_block_id"),
                    metadata.get("block_id"),
                    metadata.get("source_block_id"),
                )
                if v
            ]
        if not source_ids:
            source_ids = [canonical_id]

        title = ""
        level = 0
        if section.block_type == "heading":
            title = strip_heading_marker(section.text)
            level = int(metadata.get("heading_level") or infer_heading_level(section.text) or 1)

        left, top, right, bottom = section.bbox
        position = (section.page_number, left, right, top, bottom)
        return cls(
            id=f"sec-{index:04d}",
            text=section.text,
            block_type=section.block_type,
            page_start=section.page_number,
            page_end=section.page_number,
            bbox=section.bbox,
            order=int(metadata.get("order") or index),
            title=title,
            level=level,
            positions=[position],
            source_block_ids=source_ids,
            metadata=metadata,
        )

    def position_tag(self, zoomin: int) -> str:
        left, top, right, bottom = self.bbox
        if zoomin <= 0:
            zoomin = 1
        return f"@@{self.page_start}\t{int(left // zoomin)}\t{int(right // zoomin)}\t{int(top // zoomin)}\t{int(bottom // zoomin)}##"

    def to_section_tuple(self, parse_method: str, zoomin: int) -> tuple[str, ...]:
        tag = self.position_tag(zoomin)
        if parse_method in {"manual", "pipeline"}:
            return (self.text, self.block_type, tag)
        if parse_method == "paper":
            return (self.text + tag, self.block_type)
        return (self.text, tag)


@dataclass
class DirtyWindow:
    id: str
    sections: list[ValidatedSection]
    quality_flags: list[str]
    neighboring_titles: list[str] = field(default_factory=list)


@dataclass
class PostParseResult:
    sections: list[ValidatedSection]
    dirty_windows: list[DirtyWindow] = field(default_factory=list)

    def to_section_tuples(self, parse_method: str, zoomin: int) -> list[tuple[str, ...]]:
        return [section.to_section_tuple(parse_method, zoomin) for section in self.sections]


class LayoutRepairer(Protocol):
    def repair(self, window: DirtyWindow) -> list[dict[str, Any]]:
        ...


class NullLayoutRepairer:
    def repair(self, window: DirtyWindow) -> list[dict[str, Any]]:
        return []


class FakeLayoutRepairer:
    def __init__(self, repairs: dict[str, list[dict[str, Any]]] | None = None):
        self.repairs = repairs or {}

    def repair(self, window: DirtyWindow) -> list[dict[str, Any]]:
        return self.repairs.get(window.id, [])


class LLMBundleLayoutRepairer:
    def __init__(self, bundle, *, timeout_seconds: int = 45, max_blocks_per_call: int = 12):
        self.bundle = bundle
        self.timeout_seconds = max(1, int(timeout_seconds or 45))
        self.max_blocks_per_call = max(1, int(max_blocks_per_call or 12))

    @classmethod
    def from_runtime(
        cls,
        *,
        scope_id: str | None,
        llm_id: str | None,
        lang: str = "Chinese",
        timeout_seconds: int = 45,
        max_blocks_per_call: int = 12,
    ):
        if not scope_id:
            return None
        try:
            from api.db.joint_services.runtime_model_service import (
                get_model_config_from_provider_instance,
                get_runtime_default_model_by_type,
            )
            from api.db.services.llm_service import LLMBundle
            from common.constants import LLMType

            if llm_id:
                model_config = get_model_config_from_provider_instance(scope_id, LLMType.CHAT, llm_id)
            else:
                model_config = get_runtime_default_model_by_type(scope_id, LLMType.CHAT)
            return cls(
                LLMBundle(scope_id, model_config, lang=lang),
                timeout_seconds=timeout_seconds,
                max_blocks_per_call=max_blocks_per_call,
            )
        except Exception as exc:
            logging.warning("PaddleOCR LLM repair unavailable: %s", exc)
            return None

    def close(self) -> None:
        close = getattr(self.bundle, "close", None)
        if callable(close):
            close()

    def repair(self, window: DirtyWindow) -> list[dict[str, Any]]:
        if len(window.sections) > self.max_blocks_per_call:
            return []
        system = (
            "You repair OCR layout blocks. Preserve every source fact, number, "
            "date, unit, identifier, and source_block_id. Output strict JSON only."
        )
        user_payload = {
            "dirty_window_id": window.id,
            "quality_flags": window.quality_flags,
            "neighboring_titles": window.neighboring_titles,
            "blocks": [
                {
                    "id": section.id,
                    "source_block_ids": section.source_block_ids,
                    "page_start": section.page_start,
                    "page_end": section.page_end,
                    "block_type": section.block_type,
                    "bbox": list(section.bbox),
                    "text": section.text,
                }
                for section in window.sections
            ],
            "required_output_schema": {
                "sections": [
                    {
                        "text": "string",
                        "block_type": "heading|paragraph|list|table|formula|caption|image|unknown",
                        "source_block_ids": ["string"],
                        "title": "optional string",
                        "level": "optional integer",
                    }
                ]
            },
        }
        response_coro = asyncio.wait_for(
            self.bundle.async_chat(
                system,
                [{"role": "user", "content": json.dumps(user_payload, ensure_ascii=False)}],
                {"temperature": 0},
            ),
            timeout=self.timeout_seconds,
        )
        run_sync = getattr(self.bundle, "_run_coroutine_sync", None)
        response = run_sync(response_coro) if callable(run_sync) else asyncio.run(response_coro)
        payload = _load_json_object(response)
        sections = payload.get("sections") if isinstance(payload, dict) else None
        return sections if isinstance(sections, list) else []


class LayoutQualityGate:
    def classify(self, sections: list[ValidatedSection], pages: list[PaddleOCRPage] | None = None) -> list[tuple[ValidatedSection, list[str]]]:
        block_flags = self._block_order_flags(pages or [])
        decisions: list[tuple[ValidatedSection, list[str]]] = []
        for section in sections:
            flags: list[str] = []
            for block_id in section.source_block_ids:
                flags.extend(block_flags.get(block_id, []))
            flags.extend(self._text_flags(section))
            flags.extend(self._table_flags(section))
            decisions.append((section, sorted(set(flags))))
        self._mark_fragmented_headings(decisions)
        return decisions

    @staticmethod
    def is_dirty(flags: list[str]) -> bool:
        return bool(flags)

    @staticmethod
    def _text_flags(section: ValidatedSection) -> list[str]:
        text = section.text or ""
        stripped = text.strip()
        if not stripped:
            return ["low_information"]

        flags = []
        if "\ufffd" in text or "�" in text:
            flags.append("garbled_text")
        if _CONTROL_CHAR_RE.search(text):
            flags.append("garbled_text")

        chars = [ch for ch in text if not ch.isspace()]
        if len(chars) >= 20:
            punctuation = sum(1 for ch in chars if not ch.isalnum() and not ("\u4e00" <= ch <= "\u9fff"))
            if punctuation / len(chars) > 0.45:
                flags.append("punctuation_density_high")

        if re.search(r"(?:\b\w{1,2}\b[\s|/\\-]*){8,}", text):
            flags.append("broken_ocr_tokens")

        informative = [ch for ch in chars if ch.isalnum() or ("\u4e00" <= ch <= "\u9fff")]
        if section.block_type in {"paragraph", "list", "caption"} and len(informative) < 2:
            flags.append("low_information")
        if section.block_type == "image" and not informative:
            flags.append("low_information")
        return flags

    @staticmethod
    def _table_flags(section: ValidatedSection) -> list[str]:
        flags = []
        raw_text = str(section.metadata.get("raw_text") or section.text or "")
        if section.block_type != "table" and _looks_like_table_text(section.text):
            flags.append("table_shape_suspicious")
        if section.block_type == "table":
            markdown_shape = _markdown_table_shape(section.text)
            raw_shape = _html_table_shape(raw_text)
            if raw_shape and len(set(raw_shape)) > 1:
                flags.append("table_shape_suspicious")
            if not markdown_shape and "<table" in raw_text.lower():
                flags.append("table_shape_suspicious")
            if markdown_shape and any(width <= 1 for width in markdown_shape) and "\n" not in section.text.strip():
                flags.append("table_collapsed")
        return flags

    @staticmethod
    def _block_order_flags(pages: list[PaddleOCRPage]) -> dict[str, list[str]]:
        flags: dict[str, list[str]] = {}
        for page in pages:
            seen_orders: dict[int, str] = {}
            previous_top: float | None = None
            previous_order: int | None = None
            for block in page.blocks:
                block_id = block.id or str(block.metadata.get("canonical_block_id") or "")
                if not block_id:
                    continue
                current_flags = flags.setdefault(block_id, [])
                if block.order in seen_orders:
                    current_flags.append("block_order_conflict")
                    flags.setdefault(seen_orders[block.order], []).append("block_order_conflict")
                seen_orders[block.order] = block_id
                top = block.bbox[1]
                if previous_top is not None and previous_order is not None and block.order > previous_order and top + 20 < previous_top:
                    current_flags.append("block_order_conflict")
                previous_top = top
                previous_order = block.order
        return {block_id: sorted(set(values)) for block_id, values in flags.items() if values}

    @staticmethod
    def _mark_fragmented_headings(decisions: list[tuple[ValidatedSection, list[str]]]) -> None:
        run: list[int] = []

        def flush() -> None:
            if len(run) >= 3:
                for idx in run:
                    section, flags = decisions[idx]
                    if len(strip_heading_marker(section.text)) <= 40 and "fragmented_heading" not in flags:
                        flags.append("fragmented_heading")
            run.clear()

        for idx, (section, _) in enumerate(decisions):
            if section.block_type == "heading" and len(strip_heading_marker(section.text)) <= 40:
                run.append(idx)
            else:
                flush()
        flush()


class ContentFidelityValidator:
    def validate(self, original: list[ValidatedSection], repaired: list[ValidatedSection]) -> tuple[bool, list[str]]:
        reasons = []
        original_text = "\n".join(section.metadata.get("raw_text") or section.text for section in original)
        repaired_text = "\n".join(section.text for section in repaired)

        original_tokens = _fact_tokens(original_text)
        repaired_tokens = _fact_tokens(repaired_text)
        missing = original_tokens - repaired_tokens
        added = repaired_tokens - original_tokens
        if missing:
            reasons.append("missing_fact_tokens")
        if added:
            reasons.append("added_fact_tokens")

        coverage = _text_coverage_ratio(original_text, repaired_text)
        if coverage < 0.45:
            reasons.append("text_coverage_low")

        original_table_shape = _combined_table_shape(original)
        repaired_table_shape = _combined_table_shape(repaired)
        if original_table_shape and (not repaired_table_shape or sum(repaired_table_shape) < sum(original_table_shape)):
            reasons.append("table_shape_shrunk")

        original_ids = {block_id for section in original for block_id in section.source_block_ids}
        repaired_ids = {block_id for section in repaired for block_id in section.source_block_ids}
        if not original_ids.issubset(repaired_ids):
            reasons.append("source_block_ids_missing")

        if any(not _declared_type_parses(section) for section in repaired):
            reasons.append("declared_type_parse_failed")

        return not reasons, reasons


class PaddleOCRPostParseChain:
    def __init__(
        self,
        *,
        normalizer: PaddleOCRLayoutNormalizer | None = None,
        quality_gate: LayoutQualityGate | None = None,
        repairer: Any = None,
        fidelity_validator: ContentFidelityValidator | None = None,
        repair_enabled: bool = True,
        max_blocks_per_call: int = 12,
    ):
        self.normalizer = normalizer or PaddleOCRLayoutNormalizer()
        self.quality_gate = quality_gate or LayoutQualityGate()
        self.repairer = repairer
        self.fidelity_validator = fidelity_validator or ContentFidelityValidator()
        self.repair_enabled = repair_enabled
        self.max_blocks_per_call = max(1, int(max_blocks_per_call or 12))

    def run(self, pages: list[PaddleOCRPage]) -> PostParseResult:
        semantic_sections = self.normalizer.normalize(pages)
        sections = [ValidatedSection.from_semantic(section, idx) for idx, section in enumerate(semantic_sections)]
        decisions = self.quality_gate.classify(sections, pages)
        dirty_windows = self._dirty_windows(decisions)

        resolved: list[ValidatedSection] = []
        dirty_by_first_id = {window.sections[0].id: window for window in dirty_windows if window.sections}
        consumed: set[str] = set()
        for section, flags in decisions:
            if section.id in consumed:
                continue
            window = dirty_by_first_id.get(section.id)
            if not window:
                clean = replace(section, repair_status="clean", quality_flags=flags)
                resolved.append(clean)
                continue
            repaired_sections = self._repair_or_fallback(window)
            consumed.update(s.id for s in window.sections)
            resolved.extend(repaired_sections)

        resolved = LayoutHierarchyChunker.assign_section_paths(resolved)
        return PostParseResult(sections=resolved, dirty_windows=dirty_windows)

    def _dirty_windows(self, decisions: list[tuple[ValidatedSection, list[str]]]) -> list[DirtyWindow]:
        windows: list[DirtyWindow] = []
        current: list[ValidatedSection] = []
        current_flags: set[str] = set()

        def flush() -> None:
            if not current:
                return
            start = len(windows)
            windows.append(
                DirtyWindow(
                    id=f"dirty-{start:04d}",
                    sections=list(current),
                    quality_flags=sorted(current_flags),
                    neighboring_titles=self._neighboring_titles(decisions, current[0], current[-1]),
                )
            )
            current.clear()
            current_flags.clear()

        for section, flags in decisions:
            if self.quality_gate.is_dirty(flags):
                if len(current) >= self.max_blocks_per_call:
                    flush()
                current.append(replace(section, quality_flags=flags))
                current_flags.update(flags)
            else:
                flush()
        flush()
        return windows

    @staticmethod
    def _neighboring_titles(decisions: list[tuple[ValidatedSection, list[str]]], first: ValidatedSection, last: ValidatedSection) -> list[str]:
        titles = []
        ids = [section.id for section, _ in decisions]
        try:
            start = ids.index(first.id)
            end = ids.index(last.id)
        except ValueError:
            return titles
        for section, _ in decisions[max(0, start - 3):start]:
            if section.title:
                titles.append(section.title)
        for section, _ in decisions[end + 1:end + 4]:
            if section.title:
                titles.append(section.title)
        return titles

    def _repair_or_fallback(self, window: DirtyWindow) -> list[ValidatedSection]:
        fallback_status = "repair_skipped"
        if self.repair_enabled and self.repairer:
            try:
                repaired_payload = self.repairer.repair(window)
                repaired_sections = _sections_from_repair_payload(window, repaired_payload)
                ok, reasons = self.fidelity_validator.validate(window.sections, repaired_sections)
                if ok:
                    return [replace(section, repair_status="repaired", quality_flags=window.quality_flags) for section in repaired_sections]
                logging.info("PaddleOCR layout repair rejected for %s: %s", window.id, reasons)
                fallback_status = "repair_rejected"
            except Exception as exc:
                logging.info("PaddleOCR layout repair failed for %s: %s", window.id, exc)
                fallback_status = "repair_rejected"
        return [replace(section, repair_status=fallback_status, quality_flags=sorted(set(section.quality_flags + window.quality_flags))) for section in window.sections]


class LayoutHierarchyChunker:
    @staticmethod
    def assign_section_paths(sections: list[ValidatedSection]) -> list[ValidatedSection]:
        stack: list[tuple[int, str]] = []
        assigned: list[ValidatedSection] = []
        for section in sections:
            if section.block_type == "heading":
                level = section.level or infer_heading_level(section.text) or 1
                title = section.title or strip_heading_marker(section.text)
                stack = [(lvl, txt) for lvl, txt in stack if lvl < level]
                stack.append((level, title))
                assigned.append(replace(section, title=title, level=level, section_path=[txt for _, txt in stack]))
                continue
            path = section.section_path or [txt for _, txt in stack]
            assigned.append(replace(section, section_path=path, title=(path[-1] if path else section.title), level=(stack[-1][0] if stack else section.level)))
        return assigned

    def chunk(
        self,
        sections: list[ValidatedSection],
        *,
        chunk_token_num: int = 512,
        delimiter: str = "\n!?。；！？",
    ) -> list[dict[str, Any]]:
        chunks: list[dict[str, Any]] = []
        pending_heading: ValidatedSection | None = None
        for section in self.assign_section_paths(sections):
            if section.block_type == "heading":
                pending_heading = section
                continue

            content = section.text
            source_ids = list(section.source_block_ids)
            flags = list(section.quality_flags)
            repair_status = section.repair_status
            positions = list(section.positions)
            if pending_heading and pending_heading.section_path == section.section_path:
                heading_text = strip_heading_marker(pending_heading.text)
                if heading_text and heading_text not in content[: max(80, len(heading_text))]:
                    content = f"{heading_text}\n{content}"
                source_ids = _dedupe([*pending_heading.source_block_ids, *source_ids])
                flags = sorted(set([*pending_heading.quality_flags, *flags]))
                repair_status = _merge_repair_status([pending_heading.repair_status, repair_status])
                positions = [*pending_heading.positions, *positions]
                pending_heading = None

            split_texts = [content]
            if section.block_type not in {"table", "formula", "image"}:
                split_texts = _split_content(content, chunk_token_num, delimiter)
            for text in split_texts:
                chunks.append(
                    _chunk_dict(
                        text=text,
                        section=section,
                        source_block_ids=source_ids,
                        quality_flags=flags,
                        repair_status=repair_status,
                        positions=positions,
                    )
                )

        if pending_heading:
            chunks.append(
                _chunk_dict(
                    text=strip_heading_marker(pending_heading.text),
                    section=pending_heading,
                    source_block_ids=pending_heading.source_block_ids,
                    quality_flags=pending_heading.quality_flags,
                    repair_status=pending_heading.repair_status,
                    positions=pending_heading.positions,
                )
            )
        return chunks


def strip_heading_marker(text: str) -> str:
    stripped = (text or "").strip()
    match = _MARKDOWN_HEADING_RE.match(stripped)
    if match:
        return match.group(2).strip()
    return stripped


def infer_heading_level(text: str) -> int:
    stripped = (text or "").strip()
    match = _MARKDOWN_HEADING_RE.match(stripped)
    if match:
        return len(match.group(1))
    match = _NUMBERED_HEADING_RE.match(stripped)
    if match:
        return min(match.group(1).count(".") + 1, 6)
    if re.match(r"^(第[一二三四五六七八九十百千万0-9]+[章节篇])", stripped):
        return 1
    return 0


def _sections_from_repair_payload(window: DirtyWindow, payload: list[dict[str, Any]]) -> list[ValidatedSection]:
    repaired: list[ValidatedSection] = []
    if not payload:
        return repaired
    fallback = window.sections[0]
    for idx, item in enumerate(payload):
        if not isinstance(item, dict):
            continue
        text = str(item.get("text") or "").strip()
        if not text:
            continue
        source_ids = item.get("source_block_ids")
        if not isinstance(source_ids, list):
            source_ids = []
        source_ids = [str(v) for v in source_ids if str(v).strip()]
        section_type = str(item.get("block_type") or fallback.block_type or "paragraph")
        title = str(item.get("title") or "").strip()
        level = item.get("level")
        try:
            level = int(level)
        except (TypeError, ValueError):
            level = infer_heading_level(text) if section_type == "heading" else fallback.level
        repaired.append(
            replace(
                fallback,
                id=f"{window.id}-repair-{idx:04d}",
                text=text,
                block_type=section_type,
                title=title or (strip_heading_marker(text) if section_type == "heading" else fallback.title),
                level=level or 0,
                source_block_ids=source_ids,
                metadata={**fallback.metadata, "llm_repaired": True},
            )
        )
    return repaired


def _chunk_dict(
    *,
    text: str,
    section: ValidatedSection,
    source_block_ids: list[str],
    quality_flags: list[str],
    repair_status: str,
    positions: list[tuple[int, float, float, float, float]],
) -> dict[str, Any]:
    section_path = " > ".join(section.section_path)
    pages = sorted({int(pos[0]) for pos in positions if pos})
    normalized_positions = [[int(pos[0]), int(pos[1]), int(pos[2]), int(pos[3]), int(pos[4])] for pos in positions]
    tops = [pos[3] for pos in normalized_positions]
    chunk = {
        "content_with_weight": text,
        "embedding_text": f"Section: {section_path}\n\n{text}" if section_path else text,
        "section_path": section_path,
        "section_title": section.title or (section.section_path[-1] if section.section_path else ""),
        "section_level": section.level,
        "source_block_ids": _dedupe(source_block_ids),
        "repair_status": repair_status,
        "quality_flags": sorted(set(quality_flags)),
        "position_int": normalized_positions,
        "page_num_int": pages,
        "top_int": tops,
    }
    if section.block_type in {"table", "formula", "image"}:
        chunk["doc_type_kwd"] = section.block_type
    return chunk


def _split_content(text: str, chunk_token_num: int, delimiter: str) -> list[str]:
    if chunk_token_num <= 0:
        return [text]
    char_budget = max(300, chunk_token_num * 6)
    if len(text) <= char_budget:
        return [text]
    delimiter_chars = delimiter or "\n!?。；！？"
    pattern = "([" + re.escape(delimiter_chars) + "])"
    pieces = re.split(pattern, text)
    segments: list[str] = []
    buf = ""
    for idx in range(0, len(pieces), 2):
        part = pieces[idx]
        suffix = pieces[idx + 1] if idx + 1 < len(pieces) else ""
        candidate = part + suffix
        if not candidate:
            continue
        if buf and len(buf) + len(candidate) > char_budget:
            segments.append(buf.strip())
            buf = candidate
        else:
            buf += candidate
    if buf.strip():
        segments.append(buf.strip())
    return segments or [text]


def _merge_repair_status(statuses: Iterable[str]) -> str:
    values = set(statuses)
    for status in ("repair_rejected", "repaired", "repair_skipped", "clean"):
        if status in values:
            return status
    return "clean"


def _load_json_object(text: str) -> dict[str, Any]:
    stripped = (text or "").strip()
    if stripped.startswith("```"):
        stripped = re.sub(r"^```(?:json)?\s*", "", stripped)
        stripped = re.sub(r"\s*```$", "", stripped)
    try:
        return json.loads(stripped)
    except json.JSONDecodeError:
        start = stripped.find("{")
        end = stripped.rfind("}")
        if start >= 0 and end > start:
            return json.loads(stripped[start:end + 1])
        raise


def _fact_tokens(text: str) -> set[str]:
    return {re.sub(r"\s+", "", match.group(0)) for match in _FACT_TOKEN_RE.finditer(text or "") if match.group(0).strip()}


def _text_coverage_ratio(original: str, repaired: str) -> float:
    left = _normalize_for_coverage(original)
    right = _normalize_for_coverage(repaired)
    if not left:
        return 1.0
    if not right:
        return 0.0
    return SequenceMatcher(None, left, right).ratio()


def _normalize_for_coverage(text: str) -> str:
    if "<" in (text or "") and ">" in (text or ""):
        text = BeautifulSoup(text or "", "html.parser").get_text(" ")
    text = re.sub(r"\|?\s*:?-{3,}:?\s*\|?", " ", text or "")
    return re.sub(r"[\W_]+", "", text or "", flags=re.UNICODE).lower()


def _combined_table_shape(sections: list[ValidatedSection]) -> list[int]:
    widths: list[int] = []
    for section in sections:
        if section.block_type != "table":
            continue
        raw = str(section.metadata.get("raw_text") or "")
        raw_shape = _html_table_shape(raw)
        widths.extend(raw_shape or _markdown_table_shape(section.text))
    return widths


def _html_table_shape(text: str) -> list[int]:
    if "<table" not in (text or "").lower():
        return []
    soup = BeautifulSoup(text, "html.parser")
    table = soup.find("table")
    if table is None:
        return []
    widths = []
    for row in table.find_all("tr"):
        cells = row.find_all(["th", "td"])
        if cells:
            widths.append(len(cells))
    return widths


def _markdown_table_shape(text: str) -> list[int]:
    widths = []
    for line in (text or "").splitlines():
        stripped = line.strip()
        if not stripped.startswith("|") or not stripped.endswith("|"):
            continue
        cells = [cell.strip() for cell in stripped.strip("|").split("|")]
        if cells and all(re.fullmatch(r":?-{3,}:?", cell or "") for cell in cells):
            continue
        widths.append(len(cells))
    return widths


def _looks_like_table_text(text: str) -> bool:
    lines = [line for line in (text or "").splitlines() if line.strip()]
    if len(lines) < 2:
        return False
    tableish = sum(1 for line in lines if "|" in line or "\t" in line or len(re.split(r"\s{2,}", line.strip())) >= 3)
    return tableish >= 2


def _declared_type_parses(section: ValidatedSection) -> bool:
    text = section.text.strip()
    if section.block_type == "table":
        return bool(_markdown_table_shape(text) or "<table" in text.lower())
    if section.block_type == "formula":
        return bool(text)
    if section.block_type == "heading":
        return bool(strip_heading_marker(text))
    return bool(text)


def _dedupe(values: Iterable[str]) -> list[str]:
    result = []
    seen = set()
    for value in values:
        value = str(value)
        if value and value not in seen:
            seen.add(value)
            result.append(value)
    return result
