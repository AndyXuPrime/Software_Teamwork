#
#  Copyright 2025 The InfiniFlow Authors. All Rights Reserved.
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


def import_mistral_v2_client():
    """Return the Mistral 2.x client class.

    The repository pins ``mistralai>=2.5.1,<3.0.0``. Some 2.x wheels expose the
    class at the package top level, while the currently resolved 2.5.1 wheel
    exposes it from ``mistralai.client``. Legacy ``MistralClient`` APIs are not
    compatible with the chat/embedding calls used here.
    """

    try:
        from mistralai import Mistral

        return Mistral
    except ImportError:
        try:
            from mistralai.client import Mistral

            return Mistral
        except ImportError as exc:
            raise ImportError(
                "Mistral provider requires mistralai>=2.5.1,<3.0.0 with the Mistral 2.x client; "
                "legacy mistralai.MistralClient SDKs are not supported"
            ) from exc
