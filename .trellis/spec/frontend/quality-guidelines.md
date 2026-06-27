# Quality Guidelines

> Code quality standards for frontend development.

---

## Overview

Frontend quality checks should be exposed through package scripts so CI does
not depend on a specific build tool. Once `apps/frontend/package.json` exists,
it should provide at least:

```bash
npm run lint
npm run test
npm run build
```

Use equivalent commands if the team later chooses a different package manager,
but keep CI documented through stable script names.

---

## Forbidden Patterns

- Browser code calling backend domain services directly instead of the gateway.
- `any` for API responses or component props.
- Components that combine route orchestration, data fetching, transformation, and complex rendering in one file.
- Global state used as a cache for server data.
- Ignoring loading, empty, and error states.
- Suppressing hook or TypeScript lint rules without explaining the invariant.
- Introducing a styling, state, or validation library without updating the specs.

---

## Required Patterns

- Keep API calls in feature or shared API modules.
- Keep server state in React Query once data fetching exists.
- Use typed props and typed API functions.
- Render accessible controls with semantic HTML.
- Handle loading, empty, error, and success states for user-facing async views.
- Keep route/page components focused on composition.
- Add tests for changed behavior, not only snapshots.

---

## Testing Requirements

Use risk-based coverage:

| Change Type | Required Coverage |
|-------------|-------------------|
| Pure utility or formatter | Unit tests |
| Component behavior | Component tests |
| Data-fetching hook | Hook test with mocked gateway API or query client |
| API client change | Unit test for request/response normalization |
| Route workflow | Integration-style test where tooling supports it |
| Accessibility-sensitive component | Test keyboard and accessible names |

Testing rules:

- Prefer user-observable assertions over implementation details.
- Mock gateway APIs at the network or API-client boundary.
- Test error and empty states for async UI.
- Avoid brittle snapshots as the only coverage.

---

## Build and Lint

CI should run:

```bash
cd apps/frontend
npm ci
npm run lint
npm run test
npm run build
```

If the package manager changes, update README, CI, and this spec together.

---

## Code Review Checklist

Reviewers should check:

- [ ] Does the code belong in the chosen feature/shared directory?
- [ ] Does browser code call only the gateway API?
- [ ] Are API responses typed and normalized at the boundary?
- [ ] Are loading, empty, error, and success states handled?
- [ ] Is state scoped correctly: local, URL, server, or global?
- [ ] Are components accessible and keyboard-friendly?
- [ ] Are tests added or updated for changed behavior?
- [ ] Do lint, test, and build scripts pass?

---

## Common Mistakes

- Using the route component as a dumping ground for all feature logic.
- Promoting feature-specific UI into shared components too early.
- Adding a global store to avoid passing one or two props.
- Treating TypeScript types as runtime validation.
- Updating frontend package tooling without updating CI documentation.
