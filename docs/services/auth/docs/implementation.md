# Auth 服务实现说明

版本：v0.2
日期：2026-06-30
范围：`services/auth/` 当前实现、契约对齐、缺口和后续实现约束

## 1. 文档定位

本文档描述 `auth` 当前实现状态和后续实现约束。它只补充服务 README、OpenAPI、架构和技术选型文档，不覆盖这些上游契约。

权威来源：

| 类型 | 权威来源 | 本文档关系 |
| --- | --- | --- |
| 服务公开说明 | `docs/services/auth/README.md` | 只能补充，不能覆盖 |
| 服务 OpenAPI | `docs/services/auth/api/internal.openapi.yaml` | 只能跟随，不能另起契约 |
| Gateway 公开契约 | `docs/services/gateway/api/public.openapi.yaml` | 前端稳定契约以 gateway 为准 |
| 服务边界 | `docs/architecture/service-boundaries.md` | 必须遵守 |
| 技术基线 | `docs/architecture/technology-decisions.md` | 必须跟随 |
| 代码实现 | `services/auth/` | 本文档记录当前状态和差距 |

凡是本文档与上表文件冲突，以上游文件为准；发现冲突时，在“文档与实现出入”中记录并生成回写或实现任务。

## 2. 当前结论

| 项目 | 状态 | 说明 |
| --- | --- | --- |
| 文档状态 | active | README、数据模型和内部 OpenAPI 存在。 |
| 代码状态 | implemented | Go service、PostgreSQL repository、migrations、用户/会话内部 API、argon2id、token hash、`pgx/v5@v5.9.2` 和 `x/crypto` 安全更新已落地。 |
| 契约对齐 | aligned | Auth 内部 routes 与服务 OpenAPI 主体一致；Gateway 公开 auth routes 已有专门 handler / proxy。 |
| 数据持久化 | postgres | `AUTH_DATABASE_URL` 配置后使用 PostgreSQL；无 memory runtime。 |
| 测试状态 | covered | config、repository mapping、service crypto/session、HTTP handler 测试存在。 |
| 建议动作 | 联调 / 回写文档 | 保留 DB migration smoke 记录；Gateway/Auth/Redis full smoke 已脚本化；本地管理员和超管 seed 已补齐，生产初始化流程仍需继续完善。 |

## 3. 已实现

| 能力 | 代码位置 | 契约来源 | 验证方式 | 备注 |
| --- | --- | --- | --- | --- |
| 健康/就绪检查 | `services/auth/internal/http/server.go` | `docs/services/auth/api/internal.openapi.yaml` | `cd services/auth && go test ./...` | `/readyz` 检查 PostgreSQL。 |
| 创建用户 | `services/auth/internal/http/auth_handlers.go`、`internal/service/auth.go` | Auth OpenAPI / Gateway OpenAPI | service/http tests | 成功返回 session response。 |
| 创建会话 | `services/auth/internal/service/auth.go` | Auth OpenAPI / Gateway OpenAPI | `TestCreateSessionRejectsWrongPasswordAndRecordsFailure` | 校验密码并签发 opaque token。 |
| 查询用户和权限 | `services/auth/internal/http/server.go`、`internal/repository/postgres.go` | Auth OpenAPI | repository/http tests | 支持 user summary 和 permissions。 |
| 查询/撤销会话 | `services/auth/internal/service/auth.go` | Auth OpenAPI | `TestRevokedTokenNoLongerReturnsActiveSession` | 只保存 token hash。 |
| 当前用户资料 | `services/auth/internal/http/auth_handlers.go`、`internal/service/service.go` | Auth/Gateway OpenAPI profile paths | `cd services/auth && go test ./...` | 允许更新 `displayName`、`email`、`phone`；`email`/`phone` 可清空且不做唯一约束；写接口要求 Gateway 专用 token 和 `CallerService=gateway`。 |
| 必需改密 | `services/auth/internal/service/service.go` | Auth/Gateway OpenAPI password-change paths | `TestRequiredPasswordChangeVerifiesCurrentPasswordAndClearsFlag`、`TestRequiredPasswordChangeRejectsUserWithoutRequiredChange` | 校验当前临时密码、新密码确认、8..1024 字符策略和 `must_change_password` 当前状态，成功清除 `must_change_password` 并记录 `password.changed` 安全事件；写接口要求 Gateway 专用 token 和 `CallerService=gateway`。 |
| 管理员用户管理 | `services/auth/internal/http/auth_handlers.go`、`internal/repository/postgres.go` | Auth/Gateway OpenAPI admin-users paths | service/http/repository tests | 管理员管理 standard；super_admin 管理 standard/admin；拒绝 super_admin target 和敏感自操作；管理层级按 Auth DB 角色复核，不信任转发角色头；临时密码校验字段名保持 `temporaryPassword`，重置密码记录 `password.reset` 安全事件。 |
| 密码哈希 | `services/auth/internal/service/crypto.go` | `technology-decisions.md` | `TestPasswordHashUsesArgon2idV1PHC` | argon2id PHC 参数固定。 |
| PostgreSQL schema | `services/auth/migrations/0001_create_auth_core_tables.sql` 到 `0005_add_must_change_password.sql` | Auth 数据模型 | goose apply 需手工 | seed roles/permissions 存在；credentials 包含 `must_change_password`。 |
| 服务间 token | `services/auth/internal/http/server.go` | Auth README | handler tests | 一般内部路由校验共享 `X-Service-Token`；`/internal/v1/admin/**` 和当前用户资料/改密写接口只接受 Gateway 专用 token。 |

## 4. 未实现

| 缺口 | 文档来源 | 影响范围 | 建议任务 |
| --- | --- | --- | --- |
| 初始化管理员账号流程未形成公开 smoke | `docs/requirements-analysis/overall-requirements-analysis.md` | deploy / auth | 待确认：补 seed/admin bootstrap 文档或命令。 |
| Gateway/Auth/Redis 端到端 smoke 未进入默认 required CI | Gateway/Auth README | integration | 已新增 `bash scripts/run_issue_352_smoke.sh` 本地入口和手动 optional workflow；真实执行需要 Docker daemon、PostgreSQL、Redis 和 host-run 进程，不作为 required CI。 |
| 限流/风控不在当前代码范围 | Auth README 安全事件扩展 | security | 待确认：作为后续增强。 |

## 5. 文档与实现出入

| 出入点 | 文档要求 | 当前实现 | 风险 | 建议处理 |
| --- | --- | --- | --- | --- |
| README 状态记录 | README 曾称 `services/auth/` 代码尚未落地 | 实际已有 Go module、migrations、repository、HTTP routes；本次已回写 README | 后续若重复写实现状态，容易再次漂移 | README 只链接 implementation，当前状态在本文维护。 |
| 技术选型 pgx 版本 | 技术基线要求 PostgreSQL 服务统一使用 `pgx/v5@v5.9.2` | Auth 已迁移到 `pgx/v5@v5.9.2`，sqlc 生成代码和 repository 适配层同步更新 | 已对齐 | 后续新增 PostgreSQL 服务沿用 `pgx/v5`，不得复制旧 v4 用法。 |
| 无 DB 时 runtime | README 允许无 `AUTH_DATABASE_URL` 启动但 ready 503 | 当前 handlers 无 auth service 时业务 routes 会依赖缺失服务 | 本地误以为可用 | README/implementation 说明无 DB 仅用于进程启动检查。 |
| 用户管理契约 | OpenAPI 已新增管理员用户管理、profile 和必需改密路径 | Auth 已注册对应 handler、migration、service 和 repository 方法 | 剩余主要是前端 UI 与真实依赖联调 | 前端子任务继续接入这些 active contract。 |

## 6. MVP / mock / memory backend / 占位

| 项目 | 当前用途 | 退出条件 | 关联任务 |
| --- | --- | --- | --- |
| 无 DB 启动模式 | 本地验证 health 和配置默认值 | 真实业务联调必须配置 PostgreSQL | 无 |
| repository row tests | 不依赖真实 PostgreSQL 的 SQL mapping 校验 | 保留，同时用 env-gated migration smoke 或本地 DB smoke 证明 schema apply | 待确认 |

## 7. 运行与配置

| 项目 | 当前状态 | 缺口 |
| --- | --- | --- |
| 启动命令 | `cd services/auth && AUTH_HTTP_ADDR=:8001 go run ./cmd/server` | 业务可用需配置 DB、token secret、service token。 |
| 环境变量 | `AUTH_DATABASE_URL`、`AUTH_INTERNAL_SERVICE_TOKEN`、`AUTH_GATEWAY_ADMIN_SERVICE_TOKEN`、`AUTH_TOKEN_HASH_SECRET`、session TTL、default role、timeouts | 需要部署 secret 注入说明。 |
| PostgreSQL / migration | `migrations/0001` 到 `0005`，`sqlc.yaml`，runtime repository | 本地 PostgreSQL smoke 需补到 version 5。 |
| Redis / queue | Auth 不使用 Redis；Gateway 使用 Redis session cache | 无。 |
| Object storage / vector store / AI provider | 不涉及 | 无。 |

## 8. 测试与验证

| 验证项 | 命令或步骤 | 当前结果 | 缺口 |
| --- | --- | --- | --- |
| 单元测试 | `cd services/auth && go test ./...` | pass（2026-07-02） | 无。 |
| 集成测试 | `go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 -dir migrations postgres "$DATABASE_URL" up` | pass（2026-07-01，本地 Docker PostgreSQL 16，迁移到 version 3） | 需随 migration 版本继续更新记录。 |
| 契约测试 | HTTP handler tests + Gateway auth proxy tests | pass（2026-07-03，后端与 Gateway auth proxy 本地测试） | 未从 OpenAPI 自动生成校验。 |
| 本地 / optional CI smoke | `bash scripts/run_issue_352_smoke.sh`；手动 workflow `Auth Gateway Redis Smoke` | available（env-gated；执行会 apply Auth migration、启动 Auth/Gateway、验证 Redis 和 fake owner header capture） | 真实执行需要 Docker daemon、PostgreSQL、Redis 和 Go；默认 PR CI 只跑 skip 编译。 |

## 9. 建议任务

| 任务 | 类型 | 优先级 | 依据 | 说明 |
| --- | --- | --- | --- | --- |
| 补 Gateway/Auth/Redis 端到端 smoke | 已脚本化 | P0 | Auth 是 gateway 鉴权上游 | `scripts/run_issue_352_smoke.sh` 验证 Auth migration apply、Gateway create user/session、session cache、`/users/me`、logout、Redis 脱敏和 fake owner header capture；已提供手动 optional CI。 |
| 补管理员账号初始化说明 | 新任务 | P1 | 管理端需要管理员身份 | 形成本地和演示环境 bootstrap。 |
| 保持 pgx v5 基线 | 维护约束 | P1 | Auth 已与其他 PostgreSQL 服务统一到 pgx/v5 | 后续升级 pgx 或 sqlc 时同步更新技术基线和服务文档。 |

## 10. 最近检查记录

| 日期 | 检查人/工具 | 代码基准 | 结论 |
| --- | --- | --- | --- |
| 2026-07-03 | Codex docs watch | `develop@58fc6eb2` | 复核 #504 已在当前 develop：Auth profile、required password-change 和 admin-users 管理契约已进入 Auth/Gateway 文档；管理员用户管理由 Auth 复核 DB 角色层级，不信任转发角色头，Gateway 仅做入口鉴权、actor context 传递和响应归一化。 |
| 2026-07-03 | Codex docs watch | `develop@736acde0` | 复核 PR #520：Auth/Gateway/Redis full smoke 已由 `scripts/run_issue_352_smoke.sh` 脚本化，并新增手动 optional workflow；覆盖 Auth migration apply、Gateway create user/session、`/users/me`、logout、Redis session TTL/value/token 脱敏和 fake owner header capture。默认 PR required CI 仍不运行真实 Docker/host-run full smoke。 |
| 2026-06-29 | Codex goal | `eddf917` + working tree | Auth 实现已落地且基本对齐契约；主要剩余是 DB smoke、管理员初始化和 README 状态回写。 |
| 2026-06-30 | Codex full-day audit | `develop@92d3afc` | 复核今日 PR/issue：Auth 已保持 PostgreSQL runtime、`pgx/v5@v5.9.2` 和 `x/crypto` 安全基线；DB migration smoke 已有记录，剩余为 Gateway/Auth/Redis 端到端 smoke 与管理员初始化说明。 |
| 2026-07-02 | Codex backend task | `develop@cda73a10` + user-management working tree | Auth 已实现管理员用户管理、当前用户资料、必需改密、`must_change_password` migration/sqlc/service/handler；`go test ./...` 和 `go build ./cmd/server` 通过。 |
