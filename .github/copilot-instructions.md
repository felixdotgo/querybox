# Global Copilot Instructions (Repository Entry Point)

This file is the universal baseline for all AI models in this repository.

## 1) Purpose
- Provide one consistent instruction entrypoint for all tasks.
- Route specialized behavior to layered instruction files.
- Keep outputs precise, verifiable, and SDLC-aware.

## 2) Repository Instruction Map
- `.github/instructions/core.copilot-instructions.md`
  - Cross-language behavior, SDLC lifecycle gates, completion contract.
- `.github/instructions/go.copilot-instructions.md`
  - Go-specific coding standards.
- `.github/instructions/js.copilot-instructions.md`
  - JavaScript/TypeScript-specific coding standards.
- `.github/instructions/php.copilot-instructions.md`
  - Laravel/PHP-specific coding standards.
- `.github/instructions/system-design.copilot-instructions.md`
  - Architecture/system-design output constraints.
- `.github/instructions/project-space-template.copilot-instructions.md`
  - Placeholder template for project-specific customization.
- `.github/skills/problem-decomposition/SKILL.md`
  - AI-agent skill for execution-ready problem decomposition.
- `.github/skills/debugging-root-cause/SKILL.md`
  - AI-agent skill for evidence-based debugging and root-cause isolation.
- `.github/skills/testing-verification/SKILL.md`
  - AI-agent skill for layered verification and confidence building.
- `.github/skills/clean-code-refactor/SKILL.md`
  - AI-agent skill for safe, behavior-preserving refactoring.
- `.github/skills/security-reliability/SKILL.md`
  - AI-agent skill for trust-boundary security and reliability hardening.
- `.github/skills/delivery-sdlc-execution/SKILL.md`
  - AI-agent skill for end-to-end SDLC execution and handoff.

Skills are complementary guidance and do not override core precedence rules.

## 3) Precedence and Conflict Resolution
When multiple instructions apply, use this order:
1. Platform/system safety policies
2. User request requirements
3. `.github/instructions/core.copilot-instructions.md`
4. Relevant language/system-design overlay
5. Project-specific custom instruction (if created from template)

If rules conflict, follow higher precedence and state assumptions explicitly.

## 4) Default Operating Rules
- Match user response language unless user requests a different language.
- Keep code identifiers/comments/tests in English.
- Use evidence-based analysis; avoid inventing APIs, fields, routes, or dependencies.
- For implementation tasks, design/review tasks, and completion artifacts, follow core protocol canonical rules.

## 5) Token Efficiency Policy Routing
- Global anti token-burning rules are defined in `.github/instructions/core.copilot-instructions.md`.
- Language overlays and skills must not redefine global budgets or stop conditions.
- If any lower-level guidance conflicts with token-efficiency policy, core policy wins.

## 6) Canonical Source Matrix
- SDLC lifecycle gates: `.github/instructions/core.copilot-instructions.md`
- Completion contract and final checklist: `.github/instructions/core.copilot-instructions.md`
- Token budgets and exploration stop conditions: `.github/instructions/core.copilot-instructions.md`
- Language-specific coding rules: corresponding files under `.github/instructions/`
- Specialized workflows: corresponding files under `.github/skills/`
