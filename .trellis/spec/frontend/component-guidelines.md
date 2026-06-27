# Component Guidelines

> How React components are built in this project.

---

## Overview

Components should be small, typed, and organized by responsibility. Feature
components belong in `src/features/<feature>/components/`; domain-neutral
components belong in `src/shared/components/` only after reuse is real.

Components should render UI. Data fetching and business orchestration should
live in hooks or feature services.

---

## Component Structure

Preferred component shape:

```tsx
type KnowledgeSearchPanelProps = {
  query: string;
  isLoading?: boolean;
  onQueryChange: (query: string) => void;
};

export function KnowledgeSearchPanel({
  query,
  isLoading = false,
  onQueryChange,
}: KnowledgeSearchPanelProps) {
  return (
    <section aria-label="Knowledge search">
      {/* UI */}
    </section>
  );
}
```

Rules:

- Export named components.
- Keep props type close to the component unless shared elsewhere.
- Prefer early returns for empty, loading, and error states.
- Avoid components that both fetch data and render complex UI.
- Keep route/page components as composition layers.

---

## Props Conventions

- Use explicit props types, not `React.FC`.
- Prefer named callback props such as `onSubmit`, `onSelectFile`, `onRetry`.
- Use discriminated unions for components with mutually exclusive modes.
- Keep boolean props readable and limited.
- Do not pass entire API response objects to low-level UI components unless the component is truly domain-specific.

Avoid:

```tsx
function Button(props: any) {
  return <button {...props} />;
}
```

Prefer:

```tsx
type ButtonProps = {
  variant?: "primary" | "secondary" | "danger";
  children: React.ReactNode;
  onClick?: () => void;
  disabled?: boolean;
};
```

---

## Composition

- Prefer composition over large prop matrices.
- Use `children` for flexible content areas.
- Split complex workflows into container, section, and item components.
- Keep repeated list items stable with meaningful keys from backend IDs.
- Use shared components for generic controls only after behavior and naming are stable.

---

## Styling Patterns

The styling solution is not fixed yet. Until one is selected:

- Keep class names semantic and stable.
- Keep global styles limited to app shell, typography, resets, and design tokens.
- Do not introduce a styling library without documenting the decision.
- Avoid one-off inline style objects except for dynamic values that cannot be represented cleanly in CSS.
- Keep component layout predictable on narrow screens.

---

## Accessibility

Minimum requirements:

- Use semantic HTML before adding ARIA.
- Buttons must be `<button>`, navigation links must be links.
- Form inputs need visible labels or accessible names.
- Loading and error states must be perceivable.
- Dialogs, menus, and popovers must support keyboard interaction.
- Do not rely on color alone to communicate status.

---

## Common Mistakes

- Creating large page components that own fetching, transformation, and rendering.
- Passing backend DTOs through many component layers.
- Introducing shared components before reuse is proven.
- Hiding accessible labels for icon-only actions.
- Using array index as a key for changing lists.
