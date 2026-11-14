#!/bin/sh
# wails build が消す frontend/dist/.gitkeep を、ビルド後に復元する
cmd=$(jq -r '.tool_input.command // empty')
case "$cmd" in
  *"wails build"*|*"make build"*)
    cd "$CLAUDE_PROJECT_DIR" && git checkout -- cmd/desktop/frontend/dist/.gitkeep 2>/dev/null ;;
esac
exit 0
