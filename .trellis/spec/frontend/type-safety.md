# Type Safety

> TypeScript type-safety patterns for the frontend.

---

## Overview

Frontend code uses TypeScript. Types should document UI and API contracts
without pretending that backend responses are trustworthy at runtime.

Use compile-time types for internal code and runtime validation at API
boundaries when response shape affects user-visible behavior.

---

## Type Organization

Feature-local types:

```text
src/features/knowledge/types.ts
```

Shared cross-feature types:

```text
src/shared/types/
```

API response and request types should live near the API function that owns them:

```text
src/features/files/api/uploadFile.ts
src/features/files/api/types.ts
```

Rules:

- Keep types close to the code that owns them.
- Promote types to `shared/types` only when multiple features use them.
- Do not mirror backend database schemas directly in frontend types.
- Name API DTOs explicitly, for example `KnowledgeSearchResponse`.
- Name UI view models explicitly when transformed from API data.

---

## Runtime Validation

Use runtime validation for:

- gateway API responses with complex nested data,
- user-upload metadata,
- feature flags or remote configuration,
- data that controls permissions, rendering, or generated document workflows.

The validation library is not fixed yet. If one is introduced, document it and
use it consistently. Zod is the default candidate if the team wants a concrete
option later.

Pattern:

```ts
export type KnowledgeItem = {
  id: string;
  title: string;
  updatedAt: string;
};

export async function listKnowledgeItems(): Promise<KnowledgeItem[]> {
  const data = await gatewayClient.get("/knowledge/items");
  // validate or normalize here before returning to components
  return data.items;
}
```

---

## Common Patterns

- Use discriminated unions for workflow states:

```ts
type UploadState =
  | { status: "idle" }
  | { status: "uploading"; progress: number }
  | { status: "success"; fileId: string }
  | { status: "error"; message: string };
```

- Use `unknown` before validating external data.
- Use type guards for small validation cases.
- Use `as const` for stable literal maps.
- Prefer derived types when the source is local and stable.

---

## Forbidden Patterns

- `any` for API responses, component props, or shared utilities.
- Type assertions that silence real uncertainty, for example `response as User`.
- Non-null assertions without a preceding guard.
- Exporting massive catch-all types from `shared/types`.
- Reusing backend persistence names when frontend view models differ.

If a type assertion is unavoidable, keep it local and explain the invariant in a short comment.

---

## API Boundary Rules

- API clients return typed and normalized data.
- Components should not parse raw `fetch` responses.
- Components should not know backend error internals.
- Date/time strings from the backend should be normalized or clearly documented before display.
- Permission-sensitive fields should be treated defensively even when typed.

---

## Common Mistakes

- Trusting generated or handwritten DTO types without runtime checks at critical boundaries.
- Passing raw API objects deep into generic UI components.
- Using `Partial<T>` for form state until required fields become unclear.
- Hiding backend/frontend shape differences behind one shared type name.
