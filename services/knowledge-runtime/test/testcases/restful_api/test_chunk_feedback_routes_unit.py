#
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
import importlib.util
import sys
from pathlib import Path
from types import ModuleType, SimpleNamespace

import pytest


class _DummyManager:
    def route(self, *_args, **_kwargs):
        def decorator(func):
            return func

        return decorator


class _AwaitableValue:
    def __init__(self, value):
        self._value = value

    def __await__(self):
        async def _co():
            return self._value

        return _co().__await__()


class _DummyRequest:
    def __init__(self, body):
        self._body = body

    async def get_json(self):
        return self._body


def _run(coro):
    return asyncio.run(coro)


def _load_module(monkeypatch, body):
    repo_root = Path(__file__).resolve().parents[3]

    quart_mod = ModuleType("quart")
    quart_mod.request = _DummyRequest(body)
    monkeypatch.setitem(sys.modules, "quart", quart_mod)

    api_pkg = ModuleType("api")
    api_pkg.__path__ = [str(repo_root / "api")]
    monkeypatch.setitem(sys.modules, "api", api_pkg)

    apps_pkg = ModuleType("api.apps")
    apps_pkg.login_required = lambda func: func
    monkeypatch.setitem(sys.modules, "api.apps", apps_pkg)
    api_pkg.apps = apps_pkg

    utils_pkg = ModuleType("api.utils")
    utils_pkg.__path__ = [str(repo_root / "api" / "utils")]
    monkeypatch.setitem(sys.modules, "api.utils", utils_pkg)

    api_utils_mod = ModuleType("api.utils.api_utils")

    def add_tenant_id_to_kwargs(func):
        async def wrapper(*args, **kwargs):
            kwargs["tenant_id"] = "tenant_1"
            return await func(*args, **kwargs)

        return wrapper

    async def get_request_json():
        return await quart_mod.request.get_json()

    def get_error_data_result(message="", code=102):
        return {"code": code, "message": message}

    def get_result(data=None, code=0, message="success"):
        return {"code": code, "message": message, "data": data}

    def server_error_response(_ex):
        return {"code": 500, "message": "error"}

    api_utils_mod.add_tenant_id_to_kwargs = add_tenant_id_to_kwargs
    api_utils_mod.get_request_json = get_request_json
    api_utils_mod.get_error_data_result = get_error_data_result
    api_utils_mod.get_result = get_result
    api_utils_mod.server_error_response = server_error_response
    monkeypatch.setitem(sys.modules, "api.utils.api_utils", api_utils_mod)

    kb_service_mod = ModuleType("api.db.services.knowledgebase_service")

    class KnowledgebaseService:
        @staticmethod
        def accessible(kb_id, user_id):
            return kb_id == "kb_1"

    kb_service_mod.KnowledgebaseService = KnowledgebaseService
    monkeypatch.setitem(sys.modules, "api.db.services.knowledgebase_service", kb_service_mod)

    feedback_service_mod = ModuleType("api.db.services.chunk_feedback_service")

    class ChunkFeedbackService:
        @classmethod
        def apply_feedback(cls, tenant_id, reference, is_positive):
            return {
                "success_count": 1,
                "fail_count": 0,
                "chunk_ids": [reference["chunks"][0]["id"]],
            }

    feedback_service_mod.ChunkFeedbackService = ChunkFeedbackService
    monkeypatch.setitem(sys.modules, "api.db.services.chunk_feedback_service", feedback_service_mod)

    misc_mod = ModuleType("common.misc_utils")

    async def thread_pool_exec(func, **kwargs):
        return func(**kwargs)

    misc_mod.thread_pool_exec = thread_pool_exec
    monkeypatch.setitem(sys.modules, "common.misc_utils", misc_mod)

    spec = importlib.util.spec_from_file_location(
        "chunk_feedback_api",
        repo_root / "api" / "apps" / "restful_apis" / "chunk_feedback_api.py",
    )
    module = importlib.util.module_from_spec(spec)
    module.manager = _DummyManager()
    spec.loader.exec_module(module)
    return module


@pytest.mark.parametrize(
    "body,expected_message",
    [
        ({"thumbup": "yes", "reference": {"chunks": [{"id": "c1", "dataset_id": "kb_1"}]}}, "`thumbup` must be a boolean"),
        ({"thumbup": True}, "`reference` must be an object"),
        ({"thumbup": True, "reference": {}}, "`reference.chunks` must be a non-empty list"),
        ({"thumbup": True, "reference": {"chunks": [{"dataset_id": "kb_1"}]}}, "Each chunk requires"),
    ],
)
def test_apply_chunk_feedback_validation(monkeypatch, body, expected_message):
    module = _load_module(monkeypatch, body)
    res = _run(module.apply_chunk_feedback())
    assert res["code"] == 102
    assert expected_message in res["message"]


def test_apply_chunk_feedback_success(monkeypatch):
    body = {
        "thumbup": True,
        "reference": {
            "chunks": [
                {"id": "chunk_1", "dataset_id": "kb_1", "similarity": 0.9},
            ]
        },
    }
    module = _load_module(monkeypatch, body)
    res = _run(module.apply_chunk_feedback())
    assert res["code"] == 0
    assert res["data"]["success_count"] == 1
    assert res["data"]["chunk_ids"] == ["chunk_1"]
