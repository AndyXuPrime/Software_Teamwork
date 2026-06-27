# State Management

> How frontend state is managed.

---

## Overview

Use the smallest state scope that fits the problem. Server data should be
managed as server state, not copied into global client state.

Default choices:

- Local component state for local UI behavior.
- URL state for shareable filters, selected tabs, and navigation state.
- React Query for server state.
- Minimal global client state only for cross-cutting UI/session concerns.

---

## State Categories

| Category | Examples | Preferred Owner |
|----------|----------|-----------------|
| Local UI state | dialog open, selected row, temporary input | component or feature hook |
| URL state | search query, page, filters, selected knowledge base | router/search params |
| Server state | current user, file list, knowledge search results | React Query |
| Global client state | theme, sidebar state, lightweight auth view state | small store or context |
| Form state | upload form, document generation form | form library or component-local state |

---

## Local State

- Keep local state close to the component that owns it.
- Derive values during render when possible.
- Avoid duplicating props into state.
- Reset local state intentionally when route or resource identity changes.

---

## URL State

Use URL state for values users may bookmark, share, or return to:

- search text,
- filters,
- pagination,
- selected tab,
- selected knowledge base.

Do not put sensitive tokens or large serialized objects in the URL.

---

## Server State

Use React Query for:

- fetching gateway API data,
- caching knowledge search results,
- file upload status polling,
- document generation status polling,
- current user/session data when loaded from the backend.

Rules:

- Query keys must be stable and specific.
- Mutations must invalidate or update related queries.
- Components should not manually duplicate server data into global state.
- Loading, empty, error, and success states must all be represented in UI.

---

## When To Use Global State

Use global state only when:

- unrelated parts of the app need the same client-only state,
- prop drilling crosses multiple unrelated layout boundaries,
- the state is not server state and cannot be represented in the URL.

Examples:

- active theme,
- collapsed navigation,
- app-level notification queue,
- short-lived auth UI state.

Avoid global state for:

- API response caches,
- form drafts local to one page,
- derived values,
- values that belong in route params.

---

## Common Mistakes

- Copying React Query data into a global store.
- Storing filters in component state when they should be shareable through the URL.
- Adding global state before two unrelated areas need it.
- Keeping stale local state after resource IDs change.
- Representing loading and error states only through ad hoc booleans scattered across components.
