---
name: docs-discovery
description: "Fast documentation lookup in project docs. Activate when user request may require domain or project knowledge from documentation."
---

## When to Activate
- User asks about domain concepts, business rules, or project conventions
- Task requires understanding architecture, API contracts, or deployment
- User references documentation or asks "how does X work"
- Before making architectural decisions that existing docs may inform

## Procedure

### Fast Path (INDEX.md exists)
1. Read `docs/INDEX.md` (single file read — cheap)
2. Match user request keywords against index entries
3. Read only the matched document section(s)
4. If no match in index → fall back to Search Path

### Search Path (no INDEX.md)
1. `list_directory` on `docs/` (1 level deep)
2. Scan filenames and directory names for keyword relevance
3. If match found → read matched file(s)
4. If ambiguous → `grep` for user's key terms with `include_pattern: "docs/**"`
5. Read top 1-3 matched files (token budget)

### Deep Search (Search Path yields nothing)
1. Expand search to other common doc locations: `wiki/`, `doc/`, `documentation/`, `guides/`, `specs/`, project root `*.md`
2. `grep` across entire project for domain terms with `include_pattern: "**/*.md"`
3. Check README.md for section links or references to external docs
4. Report findings or "no documentation found for X"

### Index Generation (suggest when beneficial)
When `docs/` exists but `docs/INDEX.md` does not:
1. After completing doc search, suggest creating `docs/INDEX.md`
2. If user agrees, scan `docs/` recursively
3. For each file: extract title (first H1), key topics, path
4. Generate index table:
   - Topic/Keyword → Document path → Relevant section
   - Directory map with descriptions
5. Write `docs/INDEX.md`

### Index Maintenance
- After creating or significantly modifying any doc in `docs/`, update `docs/INDEX.md`
- After structural doc changes (move, rename, delete), rebuild affected index entries

## Decision Rules
- INDEX.md lookup > filename scan > grep search (cheapest first)
- Max 3 document reads per user request (token budget)
- Summarize relevant sections — never dump entire documents
- Prefer project docs over general knowledge for domain-specific questions
- If docs contradict code, flag the discrepancy

## Anti-patterns
- Reading all docs upfront "just in case" (token waste)
- Ignoring existing documentation and relying solely on code reading
- Not updating INDEX.md after doc changes
- Assuming doc structure without checking (e.g., assuming `docs/api/` exists)

## Output
Source documents referenced → Relevant information extracted → Answer or "not found" with search summary