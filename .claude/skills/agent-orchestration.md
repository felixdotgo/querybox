---
name: agent-orchestration
description: "Guide multi-agent task delegation for parallel execution. Activate for large tasks with independent subtrees that benefit from concurrent work."
---

## When to Activate
- Task decomposes into ≥3 independent slices (no cross-dependencies)
- Research and implementation can run in parallel
- Multiple modules need similar but independent changes
- Task involves both investigation and execution phases

## Procedure

### 1. Decomposition
1. Use `problem-decomposition` skill to break task into slices
2. Identify dependency graph between slices
3. Group slices into: **parallel-safe** (no shared files) vs **sequential** (shared state)

### 2. Agent Assignment
For each parallel-safe group:
1. Define agent scope: specific directories, files, or modules
2. Prepare full context for agent (agent has NO access to parent conversation):
   - Exact file paths to read/modify
   - Requirements and constraints
   - Coding conventions relevant to scope
   - Expected deliverable format
3. Assign non-overlapping file ownership — no two agents write to same file

### 3. Execution
1. Spawn all independent agents simultaneously
2. Keep sequential slices for main thread or spawn after dependencies complete
3. For each agent, include in prompt:
   - "Only modify files in: `<specific paths>`"
   - "Do NOT modify: `<shared files that other agents touch>`"
   - All context needed (no references to "the conversation above")

### 4. Integration
1. Collect results from all agents
2. Check for conflicts: overlapping changes, inconsistent naming, broken imports
3. Run integration verification (build, lint, test) on combined changes
4. Resolve any conflicts in main thread
5. Apply `testing-verification` skill on the integrated result

## Agent Context Template
When spawning an agent, include:
```
Task: <specific deliverable>
Files to modify: <list>
Files to read for context: <list>
Do NOT modify: <list of files other agents own>
Constraints: <relevant rules>
Output expected: <what done looks like>
```

## Decision Rules
- Only parallelize truly independent work (no shared file writes)
- Each agent must receive self-contained context
- Main thread owns coordination, conflict resolution, and integration
- Never spawn agents for tasks solvable in 1-2 tool calls
- Prefer 2-4 focused agents over many tiny agents (overhead)
- If unsure about independence → run sequentially (safer)

## Anti-patterns
- Spawning agents that write to the same files → merge conflicts
- Incomplete context in agent prompt → hallucinated paths, wrong conventions
- No integration verification after parallel work merges
- Spawning agent for trivial tasks (single file read/edit)
- Assuming agents share conversation context (they don't)

## Output
Decomposition → Agent assignments → Parallel execution → Integration verification → Combined result