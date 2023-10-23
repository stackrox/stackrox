#!/usr/bin/env bash

# Mergeswag.sh merges JSON-encoded Swagger specification files into one mongo file
# called swagger.json.

[ -z "$1" ] && echo >&2 "Please specify a folder to search for .swagger.json files" && exit 1

set -euo pipefail

folder="$1"

export TITLE="API Reference"
export VERSION="1 and 2"
export DESCRIPTION="API reference for the StackRox Security Platform"

metadata='{
  "info": {
    "title": env.TITLE,
    "version": env.VERSION,
    "description": env.DESCRIPTION,
  }
}'

find "$folder/" -name '*.swagger.json' -print0 \
	| sort -zr \
	| xargs -0 jq -s 'reduce .[] as $item ('"$metadata"'; $item * .)'
