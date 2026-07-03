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

import re
from collections.abc import Mapping, Sequence
from urllib.parse import parse_qsl, urlencode, urlsplit, urlunsplit


REDACTED = "********"

_SENSITIVE_KEY_PARTS = (
    "api_key",
    "apikey",
    "access_key",
    "secret_key",
    "client_secret",
    "private_key",
    "connection_string",
    "secret",
    "password",
    "passwd",
    "pwd",
    "token",
    "credential",
    "authorization",
    "cookie",
)

_SENSITIVE_QUERY_KEYS = (
    "api_key",
    "apikey",
    "access_key",
    "secret_key",
    "client_secret",
    "private_key",
    "secret",
    "password",
    "passwd",
    "pwd",
    "token",
    "credential",
    "authorization",
    "code",
    "signature",
    "sig",
)

_ASSIGNMENT_RE = re.compile(
    r"\b("
    r"api[_-]?key|access[_-]?key|secret[_-]?key|client[_-]?secret|"
    r"private[_-]?key|access[_-]?token|refresh[_-]?token|"
    r"password|passwd|pwd|secret|token|credential|authorization"
    r")(\s*[:=]\s*)([^,\s;&]+)",
    re.IGNORECASE,
)
_BEARER_RE = re.compile(r"\bBearer\s+[A-Za-z0-9._~+/=-]+", re.IGNORECASE)


def is_sensitive_key(key: object) -> bool:
    key_lower = str(key).lower()
    return any(part in key_lower for part in _SENSITIVE_KEY_PARTS)


def _is_sensitive_query_key(key: object) -> bool:
    key_lower = str(key).lower()
    return any(part in key_lower for part in _SENSITIVE_QUERY_KEYS)


def _netloc_without_userinfo(value: str) -> str:
    parsed = urlsplit(value)
    hostname = parsed.hostname or ""
    if ":" in hostname and not hostname.startswith("["):
        hostname = f"[{hostname}]"
    try:
        port = parsed.port
    except ValueError:
        port = None
    if port is not None:
        return f"{hostname}:{port}"
    return hostname


def redact_url(value: str) -> str:
    try:
        parsed = urlsplit(value)
    except ValueError:
        return value
    if not parsed.scheme or not parsed.netloc:
        return value

    query = ""
    if parsed.query:
        query = urlencode(
            [
                (key, REDACTED if _is_sensitive_query_key(key) else item)
                for key, item in parse_qsl(parsed.query, keep_blank_values=True)
            ],
            doseq=True,
        )

    return urlunsplit((parsed.scheme, _netloc_without_userinfo(value), parsed.path, query, ""))


def redact_text(value: str) -> str:
    redacted = redact_url(value)
    redacted = _BEARER_RE.sub(f"Bearer {REDACTED}", redacted)
    return _ASSIGNMENT_RE.sub(lambda match: f"{match.group(1)}{match.group(2)}{REDACTED}", redacted)


def sanitize_for_logging(value):
    if isinstance(value, Mapping):
        sanitized = {}
        for key, item in value.items():
            if is_sensitive_key(key):
                sanitized[key] = REDACTED if item else item
            else:
                sanitized[key] = sanitize_for_logging(item)
        return sanitized
    if isinstance(value, str):
        return redact_text(value)
    if isinstance(value, bytes):
        return f"<{len(value)} bytes>"
    if isinstance(value, Sequence) and not isinstance(value, (str, bytes, bytearray)):
        return [sanitize_for_logging(item) for item in value]
    return value
