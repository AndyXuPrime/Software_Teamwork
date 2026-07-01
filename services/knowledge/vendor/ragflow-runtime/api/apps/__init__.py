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
import logging
import os
import sys
import time
from importlib.util import module_from_spec, spec_from_file_location
from pathlib import Path
from quart import Blueprint, Quart, request, g, current_app, jsonify
from itsdangerous.url_safe import URLSafeTimedSerializer as Serializer
from quart_cors import cors
from common.constants import StatusEnum, RetCode
from api.db.db_models import close_connection, APIToken
from api.db.services import UserService
from api.utils.json_encode import CustomJSONEncoder

from quart_auth import Unauthorized as QuartAuthUnauthorized
from werkzeug.exceptions import Unauthorized as WerkzeugUnauthorized
from quart_schema import QuartSchema
from common import settings
from api.utils.api_utils import server_error_response, get_json_result
from api.constants import API_VERSION
from common.exceptions import ModelException

settings.init_settings()

__all__ = ["app"]

UNAUTHORIZED_MESSAGE = "<Unauthorized '401: Unauthorized'>"


def _unauthorized_message(error):
    if error is None:
        return UNAUTHORIZED_MESSAGE

    description = getattr(error, "description", None)
    if description:
        return description

    try:
        return repr(error)
    except Exception:
        return UNAUTHORIZED_MESSAGE


app = Quart(__name__)
app = cors(app, allow_origin="*")

# openapi supported
QuartSchema(app)

app.url_map.strict_slashes = False
app.json_encoder = CustomJSONEncoder
app.errorhandler(Exception)(server_error_response)

# Configure Quart timeouts for slow LLM responses (e.g., local Ollama on CPU)
# Default Quart timeouts are 60 seconds which is too short for many LLM backends
app.config["RESPONSE_TIMEOUT"] = int(os.environ.get("QUART_RESPONSE_TIMEOUT", 600))
app.config["BODY_TIMEOUT"] = int(os.environ.get("QUART_BODY_TIMEOUT", 600))

## convince for dev and debug
app.config["MAX_CONTENT_LENGTH"] = int(
    os.environ.get("MAX_CONTENT_LENGTH", 1024 * 1024 * 1024)
)
app.config['SECRET_KEY'] = settings.get_secret_key()
app.secret_key = settings.get_secret_key()

from functools import wraps
from typing import ParamSpec, TypeVar
from collections.abc import Awaitable, Callable, Iterable
from werkzeug.local import LocalProxy

T = TypeVar("T")
P = ParamSpec("P")

AUTH_JWT = "JWT"
AUTH_API = "API"
AUTH_BETA = "BETA"
DEFAULT_AUTH_TYPES = (AUTH_JWT, AUTH_API)


def _normalize_auth_types(auth_types=None):
    if auth_types is None:
        return set(DEFAULT_AUTH_TYPES)
    if isinstance(auth_types, str):
        return {auth_types.upper()}
    if isinstance(auth_types, Iterable):
        return {str(auth_type).upper() for auth_type in auth_types}
    return {str(auth_types).upper()}


def _load_user(auth_types=None):
    explicit_auth_types = auth_types is not None
    auth_types = _normalize_auth_types(auth_types)
    if getattr(g, "user", None) and (not explicit_auth_types or getattr(g, "auth_type", None) in auth_types):
        return g.user
    
    authorization = request.headers.get("Authorization")
    if not authorization:
        return None

    # Extract auth_token based on whether Authorization starts with "bearer" (case-insensitive)
    if authorization[:7].lower() == "bearer ":
        parts = authorization.split(maxsplit=1)
        if len(parts) < 2:
            logging.warning("Authorization header has invalid bearer format")
            return None
        auth_token = parts[1]
    else:
        auth_token = authorization

    g.user = None
    g.auth_type = None
    g.auth_error_message = None

    # Try Beta token
    if AUTH_BETA in auth_types:
        try:
            objs = APIToken.query(beta=auth_token)
            if objs:
                user = UserService.query(id=objs[0].tenant_id, status=StatusEnum.VALID.value)
                if user:
                    g.auth_type = AUTH_BETA
                    g.user = user[0]
                    return user[0]
            g.auth_error_message = 'Authentication error: API key is invalid! '
        except Exception as e_beta:
            logging.warning(f"load_user from beta token got exception {e_beta}")
            g.auth_error_message = 'Authentication error: API key is invalid!'

    # Try JWT decoding
    if AUTH_JWT in auth_types:
        try:
            jwt = Serializer(secret_key=settings.get_secret_key())
            access_token = str(jwt.loads(auth_token))

            if not access_token or not access_token.strip():
                logging.warning("Authentication attempt with empty access token")
                return None

            if len(access_token.strip()) < 32:
                logging.warning(f"Authentication attempt with invalid token format: {len(access_token)} chars")
                return None

            user = UserService.query(access_token=access_token, status=StatusEnum.VALID.value)
            if user:
                if not user[0].access_token or not user[0].access_token.strip():
                    logging.warning(f"User {user[0].email} has empty access_token in database")
                    return None
                g.auth_type = AUTH_JWT
                g.user = user[0]
                return user[0]
        except Exception as e_jwt:
            logging.warning(f"load_user from jwt got exception {e_jwt}")

    # JWT decode failed, try as api_token
    if AUTH_API in auth_types:
        try:
            objs = APIToken.query(token=auth_token)
            if objs:
                user = UserService.query(id=objs[0].tenant_id, status=StatusEnum.VALID.value)
                if user:
                    if not user[0].access_token or not user[0].access_token.strip():
                        logging.warning(f"User {user[0].email} has empty access_token in database")
                        return None
                    g.auth_type = AUTH_API
                    g.user = user[0]
                    return user[0]
                logging.warning(f"load_user: No user found for tenant_id={objs[0].tenant_id} from APIToken")
            else:
                logging.warning(f"load_user: No APIToken found for token={auth_token[:10]}...")
        except Exception as e_api_token:
            logging.warning(f"load_user from api token got exception {e_api_token}")

    return None


current_user = LocalProxy(_load_user)


def login_required(func: Callable[P, Awaitable[T]] = None, auth_types=None) -> Callable[P, Awaitable[T]]:
    """A decorator to restrict route access to authenticated users.

    This should be used to wrap a route handler (or view function) to
    enforce that only authenticated requests can access it. Note that
    it is important that this decorator be wrapped by the route
    decorator and not vice, versa, as below.

    .. code-block:: python

        @app.route('/')
        @login_required
        async def index():
            ...

    If the request is not authenticated a
    `quart.exceptions.Unauthorized` exception will be raised.

    """

    def decorator(func: Callable[P, Awaitable[T]]) -> Callable[P, Awaitable[T]]:
        @wraps(func)
        async def wrapper(*args: P.args, **kwargs: P.kwargs) -> T:
            timing_enabled = os.getenv("RAGFLOW_API_TIMING")
            t_start = time.perf_counter() if timing_enabled else None
            user = _load_user(auth_types)
            if timing_enabled:
                logging.info(
                    "api_timing login_required auth_ms=%.2f path=%s",
                    (time.perf_counter() - t_start) * 1000,
                    request.path,
                )
            if not user:  # or not session.get("_user_id"):
                if _normalize_auth_types(auth_types) == {AUTH_BETA}:
                    return get_json_result(
                        code=RetCode.DATA_ERROR,
                        message=getattr(g, "auth_error_message", None) or "Authorization is not valid!",
                    )
                raise QuartAuthUnauthorized()
            return await current_app.ensure_async(func)(*args, **kwargs)

        return wrapper

    if func is None:
        return decorator
    return decorator(func)


def search_pages_path(page_path):
    app_path_list = [path for path in page_path.glob("*_app.py") if not path.name.startswith(".")]
    restful_api_path_list = [path for path in page_path.glob("*restful_apis/*.py") if not path.name.startswith(".")]
    app_path_list.extend(restful_api_path_list)
    return app_path_list


def register_page(page_path):
    path = f"{page_path}"

    page_name = page_path.stem.removesuffix("_app")
    module_name = ".".join(page_path.parts[page_path.parts.index("api") : -1] + (page_name,))

    spec = spec_from_file_location(module_name, page_path)
    page = module_from_spec(spec)
    page.app = app
    page.manager = Blueprint(page_name, module_name)
    sys.modules[module_name] = page
    spec.loader.exec_module(page)
    page_name = getattr(page, "page_name", page_name)
    restful_api_path = "\\restful_apis\\" if sys.platform.startswith("win") else "/restful_apis/"
    url_prefix = f"/api/{API_VERSION}" if restful_api_path in path else f"/{API_VERSION}/{page_name}"

    app.register_blueprint(page.manager, url_prefix=url_prefix)
    return url_prefix


pages_dir = [
    Path(__file__).parent,
    Path(__file__).parent.parent / "api" / "apps",
    Path(__file__).parent.parent / "api" / "apps" / "restful_apis",
]

client_urls_prefix = [register_page(path) for directory in pages_dir for path in search_pages_path(directory)]


@app.errorhandler(404)
async def not_found(error):
    logging.error(f"The requested URL {request.path} was not found")
    message = f"Not Found: {request.path}"
    response = {
        "code": RetCode.NOT_FOUND,
        "message": message,
        "data": None,
        "error": "Not Found",
    }
    return jsonify(response), RetCode.NOT_FOUND


@app.errorhandler(401)
async def unauthorized(error):
    logging.warning("Unauthorized request")
    return get_json_result(code=RetCode.UNAUTHORIZED, message=_unauthorized_message(error)), RetCode.UNAUTHORIZED


@app.errorhandler(QuartAuthUnauthorized)
async def unauthorized_quart_auth(error):
    logging.warning("Unauthorized request (quart_auth)")
    return get_json_result(code=RetCode.UNAUTHORIZED, message=repr(error)), RetCode.UNAUTHORIZED


@app.errorhandler(WerkzeugUnauthorized)
async def unauthorized_werkzeug(error):
    logging.warning("Unauthorized request (werkzeug)")
    return get_json_result(code=error.code, message=error.description), RetCode.UNAUTHORIZED


@app.errorhandler(ModelException)
async def handle_model_exception(error):
    logging.warning("Forbidden request")
    return get_json_result(code=RetCode.BAD_REQUEST, message=repr(error)), 200


@app.teardown_request
def _db_close(exception):
    if exception:
        logging.exception(f"Request failed: {exception}")
    close_connection()
