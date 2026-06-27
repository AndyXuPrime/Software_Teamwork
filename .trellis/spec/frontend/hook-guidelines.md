# Hook Guidelines

> How React hooks are used in this project.

---

## Overview

Hooks should isolate stateful UI logic, data fetching, and reusable browser
behavior. Hooks must not hide broad business workflows behind unclear names.

Use React Query for server-state hooks once data fetching is implemented.
Build-tool choice does not affect hook conventions.

---

## Custom Hook Patterns

Use custom hooks when:

- multiple components share the same stateful behavior,
- a component needs complex derived state,
- browser APIs need lifecycle cleanup,
- data fetching needs a feature-specific wrapper.

Preferred shape:

```ts
export function useDebouncedValue<T>(value: T, delayMs: number): T {
  // implementation
}
```

Rules:

- Hooks must start with `use`.
- Keep hook inputs explicit.
- Return named objects for hooks with more than two return values.
- Clean up timers, subscriptions, and event listeners.
- Keep hooks deterministic and testable.

---

## Data Fetching

Feature data-fetching hooks should wrap React Query:

```ts
export function useKnowledgeSearch(query: string) {
  return useQuery({
    queryKey: ["knowledge", "search", query],
    queryFn: () => searchKnowledge(query),
    enabled: query.trim().length > 0,
  });
}
```

Rules:

- Query keys must include the feature name and all variables that affect the result.
- API functions should live outside components and hooks.
- Mutations must invalidate or update relevant queries explicitly.
- Do not call domain services directly; use the gateway API client.
- Normalize API errors before exposing them to components.

---

## Naming Conventions

- Data query hooks: `use<Resource>` or `use<Resource><Action>`, for example `useKnowledgeSearch`.
- Mutation hooks: `use<Action><Resource>`, for example `useUploadFile`.
- UI behavior hooks: `useDebouncedValue`, `useDisclosure`, `useKeyboardShortcut`.
- Avoid vague names such as `useData`, `useManager`, or `useHelper`.

---

## Side Effects

- Keep `useEffect` dependencies complete.
- Prefer event handlers for user actions instead of effect-driven workflows.
- Avoid effects that synchronize two pieces of local state when derived values would work.
- Use `AbortController` or React Query cancellation where applicable.
- Do not suppress lint rules for hooks without a comment explaining the invariant.

---

## Common Mistakes

- Fetching data directly in components without a feature hook.
- Hiding too much behavior in a hook named generically.
- Using `useEffect` to derive values that can be computed during render.
- Forgetting to invalidate React Query caches after mutations.
- Returning positional tuples with unclear meaning.
