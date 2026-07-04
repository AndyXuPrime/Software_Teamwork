from deepdoc.parser.paddleocr_post_parse import LayoutHierarchyChunker, ValidatedSection
from rag.app.naive import _tokenize_layout_chunks


def test_hierarchy_chunker_adds_inherited_context_and_keeps_display_text_clean():
    sections = [
        ValidatedSection(id="h1", text="## 1 Scope", block_type="heading", page_start=1, page_end=1, title="1 Scope", level=2, source_block_ids=["b1"]),
        ValidatedSection(id="h2", text="### 1.1 Safety", block_type="heading", page_start=1, page_end=1, title="1.1 Safety", level=3, source_block_ids=["b2"], positions=[(1, 1.0, 20.0, 5.0, 12.0)]),
        ValidatedSection(id="p1", text="The relay must trip within 20 ms.", block_type="paragraph", page_start=1, page_end=1, source_block_ids=["b3"], positions=[(1, 1.0, 20.0, 30.0, 42.0)]),
    ]

    chunks = LayoutHierarchyChunker().chunk(sections)

    assert len(chunks) == 1
    assert chunks[0]["content_with_weight"] == "1.1 Safety\nThe relay must trip within 20 ms."
    assert chunks[0]["section_path"] == "1 Scope > 1.1 Safety"
    assert chunks[0]["embedding_text"].startswith("Section: 1 Scope > 1.1 Safety")
    assert chunks[0]["source_block_ids"] == ["b2", "b3"]
    assert chunks[0]["top_int"] == [5, 30]


def test_hierarchy_chunker_keeps_table_chunks_intact():
    sections = [
        ValidatedSection(id="h1", text="## Tests", block_type="heading", page_start=1, page_end=1, title="Tests", level=2, source_block_ids=["h"]),
        ValidatedSection(
            id="t1",
            text="| Item | Value |\n| --- | --- |\n| Voltage | 220 kV |",
            block_type="table",
            page_start=1,
            page_end=1,
            source_block_ids=["t"],
        ),
    ]

    chunks = LayoutHierarchyChunker().chunk(sections, chunk_token_num=1)

    assert len(chunks) == 1
    assert chunks[0]["doc_type_kwd"] == "table"
    assert "| Voltage | 220 kV |" in chunks[0]["content_with_weight"]


def test_hierarchy_chunker_splits_oversized_paragraph_inside_same_section():
    long_text = "Sentence one. " * 40 + "Sentence two. " * 40
    sections = [
        ValidatedSection(id="h1", text="## Long", block_type="heading", page_start=1, page_end=1, title="Long", level=2, source_block_ids=["h"]),
        ValidatedSection(id="p1", text=long_text, block_type="paragraph", page_start=1, page_end=1, source_block_ids=["p"]),
    ]

    chunks = LayoutHierarchyChunker().chunk(sections, chunk_token_num=20, delimiter=".")

    assert len(chunks) > 1
    assert all(chunk["section_path"] == "Long" for chunk in chunks)


def test_layout_tokenizer_applies_child_delimiter_and_rebuilds_embedding_text():
    docs = _tokenize_layout_chunks(
        [
            {
                "content_with_weight": "alpha\nbeta",
                "embedding_text": "Section: A\n\nalpha\nbeta",
                "section_path": "A",
                "source_block_ids": ["b1"],
                "page_num_int": [1],
                "top_int": [5],
                "position_int": [[1, 1, 20, 5, 12]],
            }
        ],
        {"doc_id": "doc_1", "kb_id": "kb_1"},
        False,
        child_delimiters_pattern="\n",
    )

    assert [doc["content_with_weight"].strip() for doc in docs] == ["alpha", "beta"]
    assert all(doc["mom_with_weight"] == "alpha\nbeta" for doc in docs)
    assert docs[0]["embedding_text"].strip() == "Section: A\n\nalpha"
    assert docs[1]["embedding_text"].strip() == "Section: A\n\nbeta"
