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

from api.apps import login_required
from api.db.services.chunk_feedback_service import ChunkFeedbackService
from api.db.services.knowledgebase_service import KnowledgebaseService
from api.utils.api_utils import (
    add_tenant_id_to_kwargs,
    get_error_data_result,
    get_request_json,
    get_result,
    server_error_response,
)
from common.misc_utils import thread_pool_exec


def _validate_reference_chunks(chunks):
    if not isinstance(chunks, list) or not chunks:
        return "`reference.chunks` must be a non-empty list"

    kb_ids = set()
    for chunk in chunks:
        if not isinstance(chunk, dict):
            return "Each item in `reference.chunks` must be an object"
        chunk_id = chunk.get("id") or chunk.get("chunk_id")
        kb_id = chunk.get("dataset_id") or chunk.get("kb_id")
        if not chunk_id or not kb_id:
            return "Each chunk requires `id` (or `chunk_id`) and `dataset_id` (or `kb_id`)"
        kb_ids.add(kb_id)
    return kb_ids


@manager.route("/chunk-feedback", methods=["POST"])  # noqa: F821
@login_required
@add_tenant_id_to_kwargs
async def apply_chunk_feedback(tenant_id):
    """
    Apply user feedback to cited chunks from a retrieval or QA response.

    Requires `CHUNK_FEEDBACK_ENABLED=true` at runtime; otherwise the service
    returns success with zero updates and `"disabled": true`.
    """
    try:
        req = await get_request_json()
        if not isinstance(req, dict):
            return get_error_data_result("Request body must be a JSON object")

        thumbup = req.get("thumbup")
        if not isinstance(thumbup, bool):
            return get_error_data_result("`thumbup` must be a boolean")

        reference = req.get("reference")
        if not isinstance(reference, dict):
            return get_error_data_result("`reference` must be an object")

        validation = _validate_reference_chunks(reference.get("chunks"))
        if isinstance(validation, str):
            return get_error_data_result(validation)

        for kb_id in validation:
            if not KnowledgebaseService.accessible(kb_id=kb_id, user_id=tenant_id):
                return get_error_data_result(f"You don't own the dataset {kb_id}.")

        result = await thread_pool_exec(
            ChunkFeedbackService.apply_feedback,
            tenant_id=tenant_id,
            reference=reference,
            is_positive=thumbup,
        )
        return get_result(data=result)
    except Exception as ex:
        return server_error_response(ex)
