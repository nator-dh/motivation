#!/usr/bin/env bash
# PostToolUse hook: when a source file is edited, inject a reminder
# instructing Claude to update .claude/CODEMAP.md and .claude/ARC.md.
#
# Reads the Edit/Write/MultiEdit tool payload on stdin and, if the
# changed path matches a source pattern (and is NOT inside .claude/),
# emits a JSON document with `hookSpecificOutput.additionalContext`.
# Anything else: emits nothing -> hook is silent.

set -euo pipefail

f=$(jq -r '.tool_input.file_path // .tool_response.filePath // empty' 2>/dev/null || true)
[ -z "$f" ] && exit 0

# Loop guard: never react to edits inside .claude/ (where the docs live)
case "$f" in
  */.claude/*) exit 0 ;;
esac

is_source=0
case "$f" in
  *.go|*.yaml|*.yml|*.html) is_source=1 ;;
esac
case "$(basename "$f")" in
  Makefile) is_source=1 ;;
esac
[ "$is_source" -eq 0 ] && exit 0

# Emit additionalContext. jq builds the JSON so $f is safely escaped.
jq -nc --arg path "$f" '{
  hookSpecificOutput: {
    hookEventName: "PostToolUse",
    additionalContext: ("Source file changed: `" + $path + "`. Before ending this turn, refresh `.claude/CODEMAP.md` (concise file/module map of this repo) and `.claude/ARC.md` (architecture overview with up-to-date mermaid diagrams of components and data flow). Update only the sections affected by your change; keep diagrams current.")
  }
}'
