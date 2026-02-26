# Feature: Frontend UI

## Stack

| Layer | Library | Notes |
|-------|---------|-------|
| Framework | Vue 3 | Composition API, `<script setup>` |
| Component library | Naive UI | Themeable; use for form controls and interactive components |
| Styling | Tailwind CSS (default light theme) | Utility-first; no global dark/background overrides |
| Wails bindings | Auto-generated TypeScript | Type-safe calls to Go services |

---

## Theme & Styling Rules

- **Tailwind default light theme** — do not hardcode a global dark background or form colors in `public/style.css`.
- No inline `style="..."` attributes in Vue SFCs — use Tailwind utility classes exclusively.
- Prefer `btn-tw` / `input-tw` component classes for buttons and inputs; `input-tw` uses light background / dark text.
- Do not add global color overrides that conflict with Tailwind defaults.
- Record deliberate palette deviations in this doc and in PR descriptions.

---

## Layout Guidelines

- **Avoid `flex` for single-column full-width/height containers.** `w-full` / `h-full` or natural block flow is sufficient unless children need sibling alignment.
- Reserve `flex` for: button groups, toolbars, two-column panels with side-by-side children.
- Components expose minimal layout constraints — let parents control sizing.
- When using `flex`: always include explicit modifiers (`items-center`, `justify-between`, `gap-2`) to clarify intent and prevent accidental stretching.
- Add a comment when deviating from these guidelines (e.g. a two-column resizer that legitimately requires flex).

---

## Typography

| Property | Value |
|----------|-------|
| UI font | JetBrains Mono (Apache 2.0) — self-hosted |
| Files | `public/JetBrainsMono-Regular.ttf`, `public/JetBrainsMono-Medium.ttf` |
| Declaration | `@font-face` in `frontend/src/styles/tailwind.css` |
| Applied to | `html`, `body`, `code`, `pre`, `kbd`, `samp`, `.mono` via Tailwind `@layer base` |
| Naive UI override | `fontFamily` + `fontFamilyMono` set in `themeOverrides` in `App.vue` |
| OpenType features | `cv02–cv11` for readability; `liga`/`calt` for code ligatures |

A single font family for both UI and code contexts gives a unified developer-tool aesthetic.

---

## Icon System

**Library**: `@vicons/ionicons5` wrapped in Naive UI's `<n-icon>`.

**Golden rule**: Never import icon components directly from `@vicons/ionicons5` in Vue SFCs. Always import from `frontend/src/lib/icons.js`. Swapping the icon library requires changing only that one file.

```vue
<script setup>
import { TrashOutline } from "@/lib/icons"
</script>

<template>
  <n-icon :size="16"><TrashOutline /></n-icon>
</template>
```

---

## Adding a New View / Feature

1. Follow Naive UI for interactive controls (forms, modals, dropdowns).
2. Use Tailwind utilities for layout and spacing; never add inline styles.
3. Import icons only from `frontend/src/lib/icons.js`.
4. Subscribe to backend Wails events via `Events.On` — never emit domain events from the frontend.
5. Update `themeOverrides` in `App.vue` only for intentional theme changes; document in this file.
