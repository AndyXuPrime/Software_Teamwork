# 协作规范

本仓库采用 fork + PR 的协作方式。所有代码、文档和配置修改都必须通过
Pull Request 合入主仓库的 `develop` 分支。

## 基本原则

- 主仓库日常集成分支是 `develop`。
- 禁止向 `main` 发起日常开发 PR。
- 禁止直接向 `develop` 或 `main` push 功能、修复或文档修改。
- 必须从主仓库最新 `develop` 创建个人 fork 中的独立分支。
- PR 必须指向主仓库 `develop`。
- `main` 只用于发布，由维护者从 `develop` 做发布合并。

## 小组 Label

小组 label 是可选分类标签，不作为 PR 合并的强制条件。当前仓库使用以下小组
label：

| Label | 用途 |
|-------|------|
| `L1nggTeam` | 第一组 |
| `PrimeTeam` | 第二组 |
| `JerryTeam` | 第六组 |
| `frontend` | 前端相关 PR |
| `backend` | 后端相关 PR |

小组 label 和领域 label 都是可选分类标签。

维护者新增小组时，需要同时更新：

- GitHub 仓库 label
- 本文档的小组 label 表
- 如需自动打标，更新 [.github/labeler.json](.github/labeler.json)

PR 会根据提交账号和修改文件路径自动添加匹配 label。账号规则会匹配 PR
发起人以及 PR commit 的 GitHub author/committer login 或数字 ID。配置见
[.github/labeler.json](.github/labeler.json)。如果匹配到的 label 在仓库中不存在，
Auto Label workflow 会跳过该 label 并在日志中提示。

## 分支规则

推荐分支命名格式：

```text
<team>/<type>/<short-description>
```

示例：

```text
L1nggTeam/feat/login-page
PrimeTeam/fix/user-api-null
JerryTeam/docs/contributing-guide
```

`type` 建议与 commit type 保持一致：

- `feat`
- `fix`
- `docs`
- `style`
- `refactor`
- `test`
- `chore`
- `perf`

## Commit 规范

所有 commit 必须遵循 Conventional Commits：

```text
<type>(<scope>): <subject>
```

允许的 `type`：

| Type | 用途 |
|------|------|
| `feat` | 新功能 |
| `fix` | Bug 修复 |
| `docs` | 文档 |
| `style` | 代码格式，不改变逻辑 |
| `refactor` | 重构，不新增功能也不修复 bug |
| `test` | 测试 |
| `chore` | 构建、依赖、工具、CI/CD |
| `perf` | 性能优化 |
| `revert` | 回滚 |

好例子：

```text
feat(frontend): add login form
fix(backend): handle empty user response
docs(workflow): add fork pr rules
chore(ci): add pr guard workflow
```

坏例子：

```text
update
fix bug
wip
changes
final
```

完整规则见 [.trellis/spec/guides/commit-convention.md](.trellis/spec/guides/commit-convention.md)。

## 标准开发流程

详细命令见 [docs/git-workflow.md](docs/git-workflow.md)。

1. Fork 主仓库到个人账号。
2. 本地配置 `upstream` 指向主仓库，`origin` 指向个人 fork。
3. 拉取主仓库最新 `develop`。
4. 从 `upstream/develop` 创建个人工作分支。
5. 在个人分支提交修改。
6. 推送到个人 fork。
7. 使用 `gh pr create` 向主仓库 `develop` 发起 PR。
8. 可选添加对应小组 label。
9. 等待 CI、PR Guard、Commitlint 和 review 通过后合并。

## PR 前检查

提交 PR 前必须确认：

- 当前分支来自最新 `upstream/develop`。
- PR base 是 `develop`，不是 `main`。
- PR head 是个人 fork 的分支。
- Commit message 符合 Conventional Commits。
- 已运行相关 lint、type-check、test 或 build。

## 合并要求

合并 PR 前必须满足：

- CI 通过。
- PR Guard 通过。
- Commitlint 通过。
- 至少一名维护者 review 通过。
- 分支包含当前最新 `develop`。

推荐 GitHub Branch Protection 设置：

| Branch | 规则 |
|--------|------|
| `develop` | Require PR、Require status checks、Require branch up to date、Require approval |
| `main` | 限制发布维护者或发布机器人更新、Require status checks、禁止日常开发 PR |

维护者配置说明见 [docs/repository-settings.md](docs/repository-settings.md)。

## 维护者职责

- 不合并目标分支不是 `develop` 的日常开发 PR。
- 不合并来自主仓库分支的成员 PR。
- 发现 commit 不规范时要求 contributor rebase 或 squash 后重提。
- 发布时由维护者将 `develop` 合并到 `main`。
