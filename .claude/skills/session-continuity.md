---
name: session-continuity
description: "Manage long tasks across session limits. Auto-checkpoint progress to enable seamless resume."
---

## When to Activate
- Task decomposed into >3 slices via `problem-decomposition`
- User explicitly requests a long-running or multi-phase task
- Estimated task scope exceeds ~60% of remaining context budget

## Procedure

### Proactive Checkpointing
1. After completing each slice/phase, write checkpoint to `.claude/checkpoints/<task-slug>.md`
2. Checkpoint format:
   - **Objective**: original user request (verbatim or summarized)
   - **Completed**: list of finished slices with status and key results
   - **In Progress**: current slice, what's done, what remains
   - **Next**: ordered list of upcoming slices
   - **Decisions**: key decisions made + rationale
   - **Files Modified**: list of all changed files with brief description
   - **Risks/Blockers**: known issues, unresolved questions
   - **Timestamp**: when checkpoint was written

### Budget Awareness
- After each slice completion, assess remaining context budget
- At ~70% estimated usage: create checkpoint immediately, notify user
- At ~85% usage: finalize current step, write final checkpoint, STOP with resume instructions
- Never start a new slice if estimated completion would exceed remaining budget

### Resume Protocol
When user says "continue", "resume", or references a previous task:
1. List files in `.claude/checkpoints/` to find latest checkpoint
2. Read the checkpoint file
3. Verify current file state matches checkpoint (spot-check 2-3 key files)
4. If state matches: continue from next planned slice
5. If state diverges: report discrepancy, ask user how to proceed
6. Do NOT re-read files already summarized in checkpoint unless needed for current slice

### Checkpoint Cleanup
- On task completion, move checkpoint to `.claude/checkpoints/done/` or delete if user confirms
- Keep max 5 active checkpoints; archive oldest if exceeded

## Decision Rules
- Checkpoint after every completed slice — no exceptions for multi-slice tasks
- Prefer over-checkpointing to losing progress
- Checkpoint file should be readable by both human and AI
- Include enough context for a fresh session to resume without re-reading entire codebase

## Anti-patterns
- Skipping checkpoints for "small" remaining work that turns out to be large
- Checkpoint without file modification list (makes resume verification impossible)
- Resuming without verifying file state matches checkpoint
- Storing large code blocks in checkpoint (use file paths + line references instead)

## Output
Checkpoint file written → Resume instructions provided → Task continues or safely pauses