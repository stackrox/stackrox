#!/usr/bin/env bash

# Mergeswag.sh merges JSON-encoded Swagger specification files into one mongo file
# called swagger.json.

[ -z "$1" ] && [[ "$1" =~ ^(1|2)$ ]] && echo >&2 "Please specify a valid API version number for merged swagger.json file" && exit 1

[ -z "$2" ] && echo >&2 "Please specify at least one folder to search for .swagger.json files" && exit 1

set -euo pipefail

src=$(mktemp -d)

for folder in "${@:2}"; do
    find "$folder/" -name '*.swagger.json' -exec cp {} $src \;
done

export TITLE="API Reference"
export VERSION="$2"
export DESCRIPTION="API reference for the StackRox Security Platform"

metadata='{
  "info": {
    "title": env.TITLE,
    "version": env.VERSION,
    "description": env.DESCRIPTION,
  }
}'

find "$src" -type f -name "*" -print0 \
	| sort -zr \
	| xargs -0 jq -s 'reduce .[] as $item ('"$metadata"'; $item * .)'

rm -rf $src
