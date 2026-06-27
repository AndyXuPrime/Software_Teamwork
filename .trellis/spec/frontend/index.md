# Frontend Development Guidelines

> Engineering rules for the React + TypeScript frontend application.

---

## Overview

The frontend lives under `apps/frontend/` and talks to backend capabilities
through the gateway service. The frontend stack is React + TypeScript. The
specific build tool is intentionally not fixed yet, so specs must describe
portable React conventions and refer to package scripts such as `npm run lint`,
`npm run test`, and `npm run build`.

Frontend responsibilities:

- Provide the user interface for knowledge management workflows.
- Call gateway HTTP APIs instead of calling backend domain services directly.
- Keep server state synchronized through a dedicated data-fetching layer.
- Keep domain features isolated enough for multiple team members to work in parallel.

---

## Guidelines Index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | App, feature, shared, and API-layer layout | Active |
| [Component Guidelines](./component-guidelines.md) | Component boundaries, props, styling, accessibility | Active |
| [Hook Guidelines](./hook-guidelines.md) | Custom hooks and data-fetching hooks | Active |
| [State Management](./state-management.md) | Local, URL, global, and server state rules | Active |
| [Quality Guidelines](./quality-guidelines.md) | Lint, tests, build, and review expectations | Active |
| [Type Safety](./type-safety.md) | Type organization and runtime validation rules | Active |

---

## Pre-Development Checklist

Before changing frontend code:

- [ ] Read [Directory Structure](./directory-structure.md) for where the change belongs.
- [ ] Read [Component Guidelines](./component-guidelines.md) before adding or changing UI components.
- [ ] Read [Hook Guidelines](./hook-guidelines.md) before adding hooks or data-fetching behavior.
- [ ] Read [State Management](./state-management.md) before adding local/global/server state.
- [ ] Read [Type Safety](./type-safety.md) before changing API types or DTOs.
- [ ] Read [Quality Guidelines](./quality-guidelines.md) before declaring implementation complete.
- [ ] Run the relevant package scripts, at minimum lint and build once they exist.

---

## Frontend Principles

- Treat `gateway` as the only backend API entrypoint for browser code.
- Keep feature-specific code inside feature folders.
- Keep shared UI components domain-neutral.
- Prefer React Query for server state once data fetching is implemented.
- Keep runtime API validation at the API boundary when data shape matters.
- Do not couple components to database or backend internal schemas.

---

**Language**: Frontend spec documents are written in English.
