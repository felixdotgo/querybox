# Frontend UI

## What it is

The frontend is a Vue 3 application using Naive UI, Tailwind CSS, and Wails bindings to present the QueryBox workspace.

## Current behavior

- Composition API with `<script setup>`.
- Naive UI for interactive components.
- Tailwind for layout and styling.
- JetBrains Mono as the UI and code font family.

## Key components

- `frontend/src/App.vue`
- `frontend/src/composables/`
- `frontend/src/lib/icons.js`
- `frontend/src/styles/tailwind.css`

## How it connects to other modules

- Consumes Wails-bound backend services.
- Renders plugin-driven resource graphs and results.
- Adapts actions, result handlers, and auth flows from plugin metadata.

## Limits and roadmap

- Some UI flows still reflect strong SQL assumptions and are being generalized through the non-database workflow work.
