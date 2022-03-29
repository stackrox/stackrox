#! /bin/bash

# This script is used to pull dashboard screenshots and metrics after benchmark runs in CI.

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

mkdir -p "$DIR/captures"

ENDPOINT=${ENDPOINT:-localhost:9443}
DURATION_MIN=${DURATION_MIN:-90}

CURR_TIME=$(date +%s000) # in milliseconds
START_TIME=$(( CURR_TIME - DURATION_MIN * 60 * 1000 )) # in milliseconds
STEP="30s"

DASHBOARD_ID=${DASHBOARD_ID:-P0Ulb58nk}

DASHBOARD_FETCH_URL="https://${ENDPOINT}/api/dashboards/uid/${DASHBOARD_ID}"

# Skip panels without `.targets`, e.g., those with `"type": "row"`.
PANEL_LIST_JSON=$(curl -u admin:stackrox -sk "${DASHBOARD_FETCH_URL}" | jq '[.dashboard.panels[] | select(.targets != null) | {id: .id, title: .title, exprs: [.targets[].expr]}]')

echo "${PANEL_LIST_JSON}" | jq -c '.[]' | while read -r panel_json; do
    panel_id=$(echo "${panel_json}" | jq .id)
    panel_title_file=$(echo "${panel_json}" | jq -r .title | sed -e "s/[^A-Za-z0-9._-]/_/g" | tr "[:upper:]" "[:lower:]")

    URL="https://${ENDPOINT}/render/d-solo/${DASHBOARD_ID}/core-dashboard?orgId=1&from=$START_TIME&to=$CURR_TIME&panelId=${panel_id}&width=2000&height=2000&tz=America%2FLos_Angeles"
	curl -u admin:stackrox -sk "${URL}" > "$DIR/captures/${panel_title_file}.png"
done

mkdir -p "$DIR/rawcaptures"

kubectl -n stackrox port-forward $(kubectl -n stackrox get po -l app=monitoring -o jsonpath={.items[].metadata.name}) 9090:9090 > /dev/null &
PID=$!
sleep 5

echo "${PANEL_LIST_JSON}" | jq -c '.[]' | while read -r panel_json; do
    exprs=$(echo "${panel_json}" | jq .exprs)
    panel_title_file=$(echo "${panel_json}" | jq -r .title | sed -e "s/[^A-Za-z0-9._-]/_/g" | tr "[:upper:]" "[:lower:]")

    count=0
    echo ${exprs} | jq -rc '.[] | select(. != null)' | while read -r query; do
        count=$(( count + 1))

        # Remove metric labels because they contain dynamic variables
        query=$(echo "${query}" | sed 's/\({[A-Za-z=~$", ]*}\)//g')

        curl -s localhost:9090/api/v1/query_range \
            --data-urlencode "query=${query}" \
            --data-urlencode "start=$((START_TIME / 1000))" \
            --data-urlencode "end=$((CURR_TIME / 1000))" \
            --data-urlencode "step=${STEP}" \
            > "$DIR/rawcaptures/${panel_title_file}_${count}.json"
    done
done

kill $PID

zip -jr "$DIR/rawmetrics.zip" "$DIR/rawcaptures"
