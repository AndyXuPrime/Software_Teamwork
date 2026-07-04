from rag.llm import ocr_model
from rag.llm.ocr_model import PaddleOCROcrModel
from rag.nlp import tokenize_chunks


class FakePaddleOCRParser:
    def __init__(self, **kwargs):
        self.kwargs = kwargs
        self.outlines = [("Intro", 1)]
        self.crop_calls = []

    def crop(self, text: str, *args, **kwargs):
        self.crop_calls.append((text, args, kwargs))
        return "image-bytes", [(0, 1, 2, 3, 4)]

    def remove_tag(self, text: str):
        return text.replace("@@1\t1\t2\t3\t4##", "")


def test_paddleocr_model_exposes_pdf_parser_chunking_interfaces(monkeypatch):
    monkeypatch.setattr(ocr_model, "_load_paddleocr_parser_cls", lambda: FakePaddleOCRParser)

    model = PaddleOCROcrModel("{}", "PP-StructureV3")

    chunks = tokenize_chunks(["@@1\t1\t2\t3\t4##hello world"], {"doc_id": "doc_1"}, True, model)

    assert model.outlines == [("Intro", 1)]
    assert model._parser.crop_calls == [
        ("@@1\t1\t2\t3\t4##hello world", (), {"need_position": True})
    ]
    assert len(chunks) == 1
    assert chunks[0]["image"] == "image-bytes"
    assert chunks[0]["position_int"] == [(1, 1, 2, 3, 4)]
    assert chunks[0]["content_with_weight"] == "hello world"
