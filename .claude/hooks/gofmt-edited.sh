#!/bin/sh
# 編集された .go ファイルを gofmt で整形する(CI の gofmt チェック対策)
f=$(jq -r '.tool_input.file_path // empty')
case "$f" in
  *.go) [ -f "$f" ] && gofmt -w "$f" ;;
esac
exit 0
