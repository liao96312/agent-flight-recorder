---
name: afr
description: Record a new non-interactive Codex or Claude Code task with the local Agent Flight Recorder CLI, or list, show, verify, and clean existing AFR sessions. Use when the user asks to record, audit, inspect, or verify a coding-agent run or open its local evidence report.
---

# Agent Flight Recorder

Use the `afr` executable as a thin local wrapper. AFR records only a **new child task** launched through `afr run`; it cannot retroactively capture the current conversation or an already-running agent session.

## Before running

1. Run `afr version` and report the installed version. This plugin version requires AFR CLI `0.1.x`. If the command is missing or its version is incompatible, stop with the observed version and a clear request to install a compatible CLI; do not install it silently.
2. Resolve the workspace to an absolute path.
3. Confirm the requested host and new task. Treat `$ARGUMENTS` as the user's requested action when present.

## Record a new task

Put every AFR option before the standalone `--`. Pass the child executable and each argument separately after it; never rebuild them as a shell command string.

For Codex, launch its non-interactive mode:

```text
afr run --workspace <absolute-workspace> -- codex exec --cd <absolute-workspace> <prompt>
```

For Claude Code, launch print mode:

```text
afr run --workspace <absolute-workspace> -- claude --print <prompt>
```

Do not add approval-bypass or sandbox-bypass flags unless the user explicitly requests and understands them. After the child exits, retain the session ID printed by AFR, then run:

```text
afr show latest
afr verify latest
```

Use `afr show --json latest` or `afr verify --json latest` when structured output is needed.

## Inspect and clean sessions

- List sessions with `afr list`; add `--json` only for machine-readable output.
- Show one session with `afr show <session-id>` or `afr show latest`.
- Verify integrity with `afr verify <session-id>` or `afr verify latest`.
- Preview retention targets first with `afr clean --older-than 30d` or `afr clean --max-bytes <size>`.
- Delete only after explicit user confirmation, using the same command plus `--yes`.

## Evidence boundary

Describe only evidence AFR actually collected. Do not claim operating-system-level file or network monitoring; use the session capability matrix and explicit evidence gaps when explaining coverage.
