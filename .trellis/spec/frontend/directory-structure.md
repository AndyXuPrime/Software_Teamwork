# Directory Structure

> How the React + TypeScript frontend is organized.

---

## Overview

Frontend code lives under `apps/frontend/`. The application should use a
feature-first structure so knowledge, file, Q&A, document, and auth workflows
can evolve independently.

The specific build tool is not selected yet. Avoid build-tool-specific
directory assumptions in shared specs unless the project later chooses one.

---

## Directory Layout

```text
apps/frontend/
├── package.json
├── src/
│   ├── app/
│   ├── routes/
│   ├── features/
│   │   ├── auth/
│   │   ├── files/
│   │   ├── knowledge/
│   │   ├── qa/
│   │   └── documents/
│   ├── shared/
│   │   ├── api/
│   │   ├── components/
│   │   ├── hooks/
│   │   ├── lib/
│   │   └── types/
│   ├── assets/
│   └── styles/
└── README.md
```

Directory responsibilities:

| Directory | Responsibility |
|-----------|----------------|
| `src/app/` | Application providers, global setup, error boundaries |
| `src/routes/` | Route definitions and route-level screens |
| `src/features/<feature>/` | Feature-specific UI, hooks, API wrappers, and types |
| `src/shared/api/` | Gateway HTTP client and cross-feature API primitives |
| `src/shared/components/` | Domain-neutral reusable UI components |
| `src/shared/hooks/` | Domain-neutral reusable hooks |
| `src/shared/lib/` | Small utilities with no React dependency unless necessary |
| `src/shared/types/` | Cross-feature TypeScript types |
| `src/assets/` | Static assets |
| `src/styles/` | Global styles and design tokens |

---

## Feature Organization

Feature folders should follow this pattern when complexity requires it:

```text
src/features/knowledge/
├── api/
├── components/
├── hooks/
├── pages/
├── types.ts
└── index.ts
```

Rules:

- Keep feature-specific components inside the feature.
- Promote UI to `shared/components` only when at least two features use it.
- Keep API calls close to the feature unless they are cross-feature primitives.
- Export only intentional public feature APIs from `index.ts`.
- Do not import from another feature's private files. Use its public exports or shared code.

---

## API Layer

Browser code must call the gateway service only. It must not call `auth`,
`file`, `qa`, `knowledge`, or `document` services directly.

API responsibilities:

- Build request URLs relative to the configured gateway base URL.
- Attach authentication headers through one shared mechanism.
- Normalize errors into frontend-friendly error types.
- Validate response shapes where data is complex or user-visible.

---

## Naming Conventions

- React components use PascalCase: `KnowledgeSearchPanel.tsx`.
- Hooks use `use` prefix: `useKnowledgeSearch.ts`.
- Utility files use camelCase or kebab-case consistently within a directory.
- Feature directories use lowercase plural nouns when they represent resource collections: `files`, `documents`.
- Type files may use `types.ts` for feature-local types.
- Test files sit next to the code they cover and use `.test.ts` or `.test.tsx`.

---

## Common Mistakes

- Putting feature-specific components into `shared/components` too early.
- Calling backend domain services directly from the browser.
- Letting route components contain all data fetching, transformation, and UI logic.
- Creating circular imports between features.
- Encoding build-tool-specific assumptions before the build tool is selected.
