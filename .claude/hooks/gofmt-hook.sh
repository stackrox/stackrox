#!/bin/bash
file=$(jq -r '.tool_input.file_path')
[[ "$file" == *.go ]] && go fmt "$file" 2>/dev/null
exit 0
