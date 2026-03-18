---
name: failure-escalation
description: "Prevent endless fix loops. Escalate to re-planning or user when repeated attempts fail on the same approach."
---

## When to Activate
- Automatically active during all Implementation and debugging tasks
- Triggers when same error persists after fix attempt
- Triggers when fix introduces new errors of similar severity

## Procedure

### Level 1: Retry with Variation (max 2 attempts per approach)
1. First fix attempt fails → analyze error diff (what changed, what didn't)
2. Second attempt: adjust approach based on error diff
3. If second attempt fails → escalate to Level 2

### Level 2: Re-analyze Root Cause
1. STOP current fix approach
2. Activate `debugging-root-cause` skill
3. Re-examine: is the error where we think it is?
4. Check assumptions: correct API? correct types? correct file paths?
5. Identify alternative approach
6. Try alternative approach (returns to Level 1 with new approach)

### Level 3: Re-plan (after 2 different approaches fail)
1. STOP implementation entirely
2. Switch to Planning mode
3. Activate `problem-decomposition` skill
4. Re-decompose the problem with new constraints learned from failures
5. Present new plan to user before proceeding
6. If user approves → execute new plan (returns to Level 1)

### Level 4: Escalate to User (after re-plan still fails)
1. STOP all attempts
2. Report to user:
   - Original objective
   - Approaches tried and why each failed
   - Root cause hypothesis (best guess with confidence level)
   - What information or access would help
   - Suggested next steps for user
3. Wait for user guidance before proceeding

## Tracking
Maintain mental count per task:
- `approach_attempts`: resets when switching to new approach
- `approaches_tried`: increments on each new approach
- `replans`: increments on each Level 3 trigger

## Decision Rules
- Never retry identical approach more than twice
- Each new approach must differ meaningfully from previous ones
- Re-planning must incorporate lessons from failed approaches
- Transparency: always tell user when escalating levels
- Prefer asking user over making increasingly speculative fixes

## Anti-patterns
- Retrying same fix with trivial variations (e.g., only changing variable names)
- Suppressing errors instead of fixing root cause
- Abandoning working code to "start fresh" without analysis
- Silent escalation without informing user what was tried
- Removing functionality to make errors disappear

## Output
Level reached → Approaches tried → Current status → Next action or user question