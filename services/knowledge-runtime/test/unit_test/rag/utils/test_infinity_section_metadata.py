import importlib.util
import json
import sys
import types
from pathlib import Path

import pandas as pd


def _install_infinity_stubs(monkeypatch):
    fake_infinity = types.ModuleType("infinity")
    fake_common = types.ModuleType("infinity.common")
    fake_errors = types.ModuleType("infinity.errors")
    fake_index = types.ModuleType("infinity.index")

    class InfinityException(Exception):
        def __init__(self, *args, error_code=0):
            super().__init__(*args)
            self.error_code = error_code

    fake_common.InfinityException = InfinityException
    fake_common.SortType = types.SimpleNamespace(Asc="asc", Desc="desc")
    fake_common.ConflictType = types.SimpleNamespace(Ignore="ignore")
    fake_errors.ErrorCode = types.SimpleNamespace(TABLE_NOT_EXIST=3022)
    fake_index.IndexInfo = lambda *args, **kwargs: (args, kwargs)
    fake_index.IndexType = types.SimpleNamespace(FullText="fulltext", Secondary="secondary", Hnsw="hnsw")
    fake_infinity.common = fake_common

    fake_decorator = types.ModuleType("common.decorator")
    fake_decorator.singleton = lambda cls: cls
    fake_settings = types.ModuleType("common.settings")
    fake_settings.INFINITY = {}
    fake_settings.docStoreConn = None
    fake_nlp = types.ModuleType("rag.nlp")
    fake_nlp.is_english = lambda *_args, **_kwargs: False

    for name, module in {
        "infinity": fake_infinity,
        "infinity.common": fake_common,
        "infinity.errors": fake_errors,
        "infinity.index": fake_index,
        "common.decorator": fake_decorator,
        "common.settings": fake_settings,
        "rag.nlp": fake_nlp,
    }.items():
        monkeypatch.setitem(sys.modules, name, module)


def _load_infinity_module(monkeypatch):
    _install_infinity_stubs(monkeypatch)
    runtime_root = Path(__file__).resolve().parents[4]
    source = runtime_root / "rag" / "utils" / "infinity_conn.py"
    spec = importlib.util.spec_from_file_location("test_infinity_conn_module", source)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


def test_infinity_compacts_section_metadata_into_extra_and_drops_embedding_text(monkeypatch):
    module = _load_infinity_module(monkeypatch)
    row = {
        "id": "chunk_1",
        "content": "body",
        "section_path": "Root > Relay",
        "source_block_ids": ["p1-b1"],
        "section_path_tks": "root relay",
        "embedding_text": "Section: Root > Relay\n\nbody",
    }

    module.InfinityConnection._compact_extra_fields(row, {"id", "content", "extra"})

    assert "section_path" not in row
    assert "source_block_ids" not in row
    assert "section_path_tks" not in row
    assert "embedding_text" not in row
    assert json.loads(row["extra"]) == {
        "section_path": "Root > Relay",
        "section_path_tks": "root relay",
        "source_block_ids": ["p1-b1"],
    }


def test_infinity_hydrates_section_metadata_from_extra(monkeypatch):
    module = _load_infinity_module(monkeypatch)
    conn = module.InfinityConnection.__new__(module.InfinityConnection)
    frame = pd.DataFrame(
        [
            {
                "id": "chunk_1",
                "content": "body",
                "extra": json.dumps(
                    {
                        "section_path": "Root > Relay",
                        "section_level": 2,
                        "source_block_ids": ["p1-b1"],
                    }
                ),
            }
        ]
    )

    fields = conn.get_fields(frame, ["content_with_weight", "section_path", "section_level", "source_block_ids", "embedding_text"])

    assert fields["chunk_1"]["content_with_weight"] == "body"
    assert fields["chunk_1"]["section_path"] == "Root > Relay"
    assert fields["chunk_1"]["section_level"] == 2
    assert fields["chunk_1"]["source_block_ids"] == ["p1-b1"]
    assert fields["chunk_1"]["embedding_text"] is None


def test_infinity_select_and_match_fields_avoid_missing_section_columns(monkeypatch):
    module = _load_infinity_module(monkeypatch)
    conn = module.InfinityConnection.__new__(module.InfinityConnection)

    assert set(conn.convert_select_fields(["section_path", "embedding_text", "content_with_weight"])) == {"extra", "content"}
    assert conn.convert_matching_field("section_title_tks^12") == "content@ft_content_rag_coarse^12"
