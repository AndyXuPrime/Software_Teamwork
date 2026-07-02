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
from contextlib import AbstractContextManager

from api.utils.gateway_tenant_provisioning import ensure_gateway_tenant_with_store


def ensure_gateway_tenant(external_id):
    from api.db.init_data import init_env_default_models_for_tenant

    return ensure_gateway_tenant_with_store(
        external_id,
        PeeweeGatewayTenantStore(),
        defaults=_runtime_defaults(),
        id_factory=_new_runtime_id,
        model_initializer=init_env_default_models_for_tenant,
    )


class PeeweeGatewayTenantStore:
    def atomic(self) -> AbstractContextManager:
        from api.db.db_models import DB

        return DB.atomic()

    def get_user(self, runtime_id):
        from api.db.services.user_service import UserService

        return UserService.filter_by_id(runtime_id)

    def create_user(self, runtime_id, payload):
        import peewee
        from api.db.services.user_service import UserService

        try:
            UserService.insert(**payload)
        except peewee.IntegrityError:
            pass
        return self.get_user(runtime_id)

    def update_user(self, runtime_id, payload):
        from api.db.services.user_service import UserService

        UserService.update_by_id(runtime_id, payload)

    def get_tenant(self, runtime_id):
        from api.db.services.user_service import TenantService

        exists, tenant = TenantService.get_by_id(runtime_id)
        return tenant if exists else None

    def create_tenant(self, runtime_id, payload):
        import peewee
        from api.db.services.user_service import TenantService

        try:
            TenantService.insert(**payload)
        except peewee.IntegrityError:
            pass
        return self.get_tenant(runtime_id)

    def update_tenant(self, runtime_id, payload):
        from api.db.services.user_service import TenantService

        TenantService.update_by_id(runtime_id, payload)

    def get_user_tenant(self, user_id, tenant_id):
        from api.db.db_models import UserTenant

        return UserTenant.get_or_none(
            UserTenant.user_id == user_id,
            UserTenant.tenant_id == tenant_id,
        )

    def create_user_tenant(self, user_id, tenant_id, payload):
        import peewee
        from api.db.services.user_service import UserTenantService

        try:
            UserTenantService.insert(**payload)
        except peewee.IntegrityError:
            pass
        return self.get_user_tenant(user_id, tenant_id)

    def update_user_tenant(self, relation_id, payload):
        from api.db.services.user_service import UserTenantService

        UserTenantService.update_by_id(relation_id, payload)


def _runtime_defaults():
    from common import settings

    return {
        "chat": settings.CHAT_MDL,
        "embedding": settings.EMBEDDING_MDL,
        "rerank": settings.RERANK_MDL,
        "asr": settings.ASR_MDL,
        "image2text": settings.IMAGE2TEXT_MDL,
        "parsers": settings.PARSERS,
    }


def _new_runtime_id():
    from common.misc_utils import get_uuid

    return get_uuid()
