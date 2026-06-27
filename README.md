# Software Teamwork

A full-stack project managed with [Trellis](https://github.com/trellis-workflow).

## Getting Started

> Project setup instructions to be added.

## Conventions

### Collaboration

- Work in a personal fork unless you are repository owner `Sakayori-Iroha-168`.
- Create a dedicated branch from the latest `upstream/develop`, or `origin/develop` if you are the repository owner.
- Open PRs to `develop` only.
- Keep `main` for release merges from `develop`.

Full guide: [CONTRIBUTING.md](CONTRIBUTING.md)

GitHub CLI workflow: [docs/git-workflow.md](docs/git-workflow.md)

Repository guard settings: [docs/repository-settings.md](docs/repository-settings.md)

### Commit Messages

All commits follow [Conventional Commits](https://www.conventionalcommits.org/). Quick reference:

```
<type>(<scope>): <subject>
```

Common types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `perf`, `revert`

Full spec: [`.trellis/spec/guides/commit-convention.md`](.trellis/spec/guides/commit-convention.md)

### Code Style

- Backend: [`.trellis/spec/backend/`](.trellis/spec/backend/)
- Frontend: [`.trellis/spec/frontend/`](.trellis/spec/frontend/)

## Project Structure

```
.trellis/   # Workflow, tasks, and spec
```
