---
name: afr
description: Record a new non-interactive Codex or Claude Code task with the local Agent Flight Recorder CLI, or list, show, verify, and clean existing AFR sessions. Use when the user asks to record, audit, inspect, or verify a coding-agent run or open its local evidence report.
---

# Agent Flight Recorder

Use the `afr` executable as the single recorder. Trusted Codex Desktop hooks automatically record future turns after plugin installation and a full Desktop restart. AFR never retroactively imports earlier messages. `afr run` remains the explicit wrapper for new non-interactive Codex or Claude Code child tasks.

## Codex Desktop capture

- The plugin sends `SessionStart`, `UserPromptSubmit`, `PreToolUse`, `PostToolUse`, and `Stop` to `afr hook --host codex`.
- `resume` and `compact` reuse the mapped AFR session; `startup` and `clear` switch according to the host lifecycle.
- `Stop` creates an `idle` checkpoint and refreshes the report; it is not a reliable SessionEnd and must not be described as `completed`.
- Desktop capture starts only after the plugin is installed, trusted in `/hooks`, and the Desktop app is fully restarted. A resumed task records only later turns.
- Hosted tools that do not emit local Hook events remain `not_observable`.

## Before running

1. Run `afr version` and report the installed version. This plugin version requires AFR CLI `0.1.x`. If the command is missing or its version is incompatible, stop with the observed version and a clear request to install a compatible CLI; do not install it silently.
2. Resolve the workspace to an absolute path.
3. Confirm the requested host and new task. Treat `$ARGUMENTS` as the user's requested action when present.
4. For Claude Code, require an existing Claude login or compatible API provider authentication. AFR does not configure or persist credentials; `claude --version` alone does not prove authentication. If the child returns `Not logged in`, report the completed failed session and stop.

AFR automatically honors a workspace-root `.afrignore` and strict `.afr.json` custom RE2 redaction rules. If either file is invalid, report the configuration error; do not bypass or rewrite it unless the user asks.

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
afr show --open latest
afr verify latest
```

Use `afr show --json latest` or `afr verify --json latest` when structured output is needed.

## Inspect and clean sessions

- List sessions with `afr list`; add `--json` only for machine-readable output.
- Show one session with `afr show <session-id>` or `afr show latest`; add `--open` to open its local HTML report.
- Verify integrity with `afr verify <session-id>` or `afr verify latest`.
- Preview retention targets first with `afr clean --older-than 30d` or `afr clean --max-bytes <size>`.
- Resumable Codex Desktop Hook sessions are protected and skipped by cleanup.
- Without `--yes`, clean is always preview-only. Delete only after explicit user confirmation, using the same command plus `--yes`.

## Evidence boundary

Describe only evidence AFR actually collected. Do not claim operating-system-level file or network monitoring; use the session capability matrix and explicit evidence gaps when explaining coverage.
