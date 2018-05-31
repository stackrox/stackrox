#!/usr/bin/env bash

# Mergeswag.sh merges JSON-encoded Swagger specification files into one mongo file
# called swagger.json.

[ -z "$1" ] && echo "Please specify a folder to search for .swagger.json files" && exit 1

set -euxo pipefail

folder="$1"

specs=$(echo "$folder/*.swagger.json" | sort)

cur="$folder/cur.json"
echo "{}" > "$cur"

mongo="$folder/mongo.json"

for next in $specs
do
    jq -s '.[0] * .[1]' "$cur" "$next" > "$mongo"
    mv "$mongo" "$cur"
done

export TITLE="API Reference"
export VERSION="1"
export DESCRIPTION="API reference for the StackRox Security Platform"
export CONTACT_EMAIL="support@stackrox.com"
export LICENSE_NAME="All Rights Reserved"
export LICENSE_URL="http://www.stackrox.com/"
export HOST="localhost:3000"

cat "$cur" \
    | jq .'info.title=env.TITLE' \
    | jq .'info.version=env.VERSION' \
    | jq .'info.description=env.DESCRIPTION' \
    | jq .'info.contact.email=env.CONTACT_EMAIL' \
    | jq .'info.license.name=env.LICENSE_NAME' \
    | jq .'info.license.url=env.LICENSE_URL' \
    | jq .'host=env.HOST' \
    > "$folder/swagger.json"

