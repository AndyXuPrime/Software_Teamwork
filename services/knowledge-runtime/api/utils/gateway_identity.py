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
import hashlib


MAX_RUNTIME_ID_LENGTH = 32
GATEWAY_ID_PREFIX = "gw_"


def normalize_gateway_principal_id(external_id) -> str:
    """Map gateway IDs into RAGFlow's 32-character user/tenant id columns."""

    value = str(external_id or "").strip()
    if not value:
        return ""
    if len(value) <= MAX_RUNTIME_ID_LENGTH:
        return value
    digest = hashlib.sha256(value.encode("utf-8")).hexdigest()
    return f"{GATEWAY_ID_PREFIX}{digest[:MAX_RUNTIME_ID_LENGTH - len(GATEWAY_ID_PREFIX)]}"
