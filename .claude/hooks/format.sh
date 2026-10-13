#!/bin/bash
# PostToolUse: formatea el archivo editado con la herramienta disponible. Nunca falla.
f=$(python3 -I -c 'import json,sys;print(json.load(sys.stdin).get("tool_input",{}).get("file_path",""))')
[ -f "$f" ] || exit 0
case "$f" in
  *.ts|*.tsx|*.js|*.jsx|*.json|*.css|*.md)
    [ -x node_modules/.bin/prettier ] && node_modules/.bin/prettier --write "$f" >/dev/null 2>&1 ;;
  *.py)
    if command -v ruff >/dev/null; then ruff format "$f" >/dev/null 2>&1
    elif command -v black >/dev/null; then black -q "$f" >/dev/null 2>&1; fi ;;
  *.go) command -v gofmt >/dev/null && gofmt -w "$f" ;;
esac
exit 0
