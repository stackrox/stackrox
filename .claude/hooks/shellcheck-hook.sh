#!/bin/bash
# Shell lint hook — runs shellcheck with same flags as CI (`make shell-style`)
file=$(jq -r '.tool_input.file_path')
[[ "$file" != *.sh ]] && exit 0
[[ ! -f "$file" ]] && exit 0
shellcheck --norc -P SCRIPTDIR -x "$file" 2>&1 | head -15
exit 0
