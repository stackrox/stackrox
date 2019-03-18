#!/usr/bin/env bash

# Mergeswag.sh merges JSON-encoded Swagger specification files into one mongo file
# called swagger.json.

[ -z "$1" ] && echo "Please specify a folder to search for .swagger.json files" && exit 1

set -euo pipefail

folder="$1"

export TITLE="API Reference"
export VERSION="1"
export DESCRIPTION="API reference for the StackRox Security Platform"
export CONTACT_EMAIL="support@stackrox.com"
export LICENSE_NAME="All Rights Reserved"
export LICENSE_URL="https://www.stackrox.com/"

metadata='{
  "info": {
    "title": env.TITLE,
    "version": env.VERSION,
    "description": env.DESCRIPTION,
    "contact": {
      "email": env.CONTACT_EMAIL
    },
    "license": {
      "name": env.LICENSE_NAME,
      "url": env.LICENSE_URL
    }
  }
}'

find "$folder/" -name '*.swagger.json' -print0 \
	| sort -zr \
	| xargs -0 jq -s 'reduce .[] as $item ('"$metadata"'; $item * .)' \
		> "$folder/swagger.json"
