#! /bin/bash

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

mkdir -p "$DIR/captures"

ENDPOINT=${ENDPOINT:-localhost:9443}
DURATION_MIN=${DURATION_MIN:-60}

CURR_TIME=$(date +%s000) # in milliseconds
START_TIME=$(( CURR_TIME - DURATION_MIN * 60 * 1000 )) # in milliseconds

DASHBOARD_FETCH_URL="https://${ENDPOINT}/api/dashboards/uid/Q0AZXCdZk"

PANEL_LIST_JSON=$(curl -u admin:stackrox -sk "${DASHBOARD_FETCH_URL}" | jq '[.dashboard.panels[] | {id: .id, title: .title}]')

echo "${PANEL_LIST_JSON}" | jq -c '.[]' | while read -r panel_json; do
    panel_id=$(echo "${panel_json}" | jq .id)
    panel_title_file=$(echo "${panel_json}" | jq -r .title | sed -e "s/[^A-Za-z0-9._-]/_/g" | tr "[:upper:]" "[:lower:]")

    URL="https://${ENDPOINT}/render/d-solo/Q0AZXCdZk/core-dashboard?orgId=1&from=$START_TIME&to=$CURR_TIME&panelId=${panel_id}&width=2000&height=2000&tz=America%2FLos_Angeles"
	curl -u admin:stackrox -sk "${URL}" > "$DIR/captures/${panel_title_file}.png"
done
