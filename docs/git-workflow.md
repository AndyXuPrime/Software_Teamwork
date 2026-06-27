# GitHub CLI 工作流

本文档给出本项目推荐的 `gh` CLI 操作流程。普通成员必须通过个人 fork 的
独立分支向主仓库 `develop` 发起 Pull Request。有且只有仓库主
`Sakayori-Iroha-168` 可以直接在主仓库内创建独立分支并发起 PR。

以下示例中：

- `Sakayori-Iroha-168/Software_Teamwork` 是主仓库。
- `YOUR_NAME/Software_Teamwork` 是你的个人 fork。
- `L1nggTeam`、`PrimeTeam`、`JerryTeam` 是可选小组 label。

## 1. 登录 GitHub CLI

```bash
gh auth login
gh auth status
```

## 2. Fork 主仓库

普通成员必须 fork：

```bash
gh repo fork Sakayori-Iroha-168/Software_Teamwork --remote --clone=false
```

如果已经 fork 过，可以跳过这一步。仓库主不需要 fork。

## 3. 配置 Remote

普通成员推荐配置：

确认 remote：

```bash
git remote -v
```

推荐配置：

```bash
git remote set-url origin git@github.com:YOUR_NAME/Software_Teamwork.git
git remote add upstream git@github.com:Sakayori-Iroha-168/Software_Teamwork.git
```

如果 `upstream` 已存在：

```bash
git remote set-url upstream git@github.com:Sakayori-Iroha-168/Software_Teamwork.git
```

最终应满足：

```text
origin    -> 你的个人 fork
upstream  -> 主仓库
```

仓库主可以只保留：

```text
origin    -> 主仓库
```

## 4. 从最新 develop 创建分支

普通成员：

```bash
git fetch upstream
git switch -c L1nggTeam/feat/login-page upstream/develop
```

仓库主：

```bash
git fetch origin
git switch -c L1nggTeam/feat/login-page origin/develop
```

不要从 `main`、本地旧分支或主仓库临时分支创建开发分支。

## 5. 提交修改

```bash
git status
git add .
git commit -m "feat(frontend): add login page"
```

Commit message 必须遵循 [Conventional Commits](../.trellis/spec/guides/commit-convention.md)。

## 6. 推送到个人 fork

普通成员：

```bash
git push -u origin L1nggTeam/feat/login-page
```

仓库主推送到主仓库同名分支：

```bash
git push -u origin L1nggTeam/feat/login-page
```

## 7. 创建 PR 到主仓库 develop

普通成员：

```bash
gh pr create \
  --repo Sakayori-Iroha-168/Software_Teamwork \
  --base develop \
  --head YOUR_NAME:L1nggTeam/feat/login-page \
  --title "feat(frontend): add login page" \
  --body-file .github/pull_request_template.md
```

仓库主：

```bash
gh pr create \
  --repo Sakayori-Iroha-168/Software_Teamwork \
  --base develop \
  --head Sakayori-Iroha-168:L1nggTeam/feat/login-page \
  --title "feat(frontend): add login page" \
  --body-file .github/pull_request_template.md
```

注意：

- `--base` 必须是 `develop`。
- 普通成员 `--head` 必须是 `YOUR_NAME:<branch>`，也就是个人 fork 中的分支。
- 只有仓库主可以使用 `Sakayori-Iroha-168:<branch>` 作为同仓库 PR head。
- 可选使用 `gh pr edit <PR_NUMBER> --add-label <LABEL>` 添加小组 label。

## 8. PR 前同步最新 develop

如果主仓库 `develop` 更新了，需要 rebase：

普通成员：

```bash
git fetch upstream
git rebase upstream/develop
git push --force-with-lease
```

仓库主：

```bash
git fetch origin
git rebase origin/develop
git push --force-with-lease
```

禁止使用普通 `--force`。只使用 `--force-with-lease`。

## 9. 查看 PR 状态

```bash
gh pr status
gh pr checks <PR_NUMBER> --repo Sakayori-Iroha-168/Software_Teamwork
gh pr view --web
```

## 10. 常见错误

### PR 目标分支选成 main

关闭该 PR，重新向 `develop` 发起 PR：

```bash
gh pr close <PR_NUMBER> --repo Sakayori-Iroha-168/Software_Teamwork
gh pr create --repo Sakayori-Iroha-168/Software_Teamwork --base develop
```

### 添加小组 label

```bash
gh pr edit <PR_NUMBER> \
  --repo Sakayori-Iroha-168/Software_Teamwork \
  --add-label L1nggTeam
```

### 分支落后 develop

普通成员：

```bash
git fetch upstream
git rebase upstream/develop
git push --force-with-lease
```

仓库主：

```bash
git fetch origin
git rebase origin/develop
git push --force-with-lease
```

### Commit message 不规范

修改最近一次 commit：

```bash
git commit --amend -m "fix(backend): handle empty user response"
git push --force-with-lease
```

修改多个 commit：

```bash
git fetch upstream
git rebase -i upstream/develop
git push --force-with-lease
```
