#! /bin/bash

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

mkdir -p "$DIR/captures"

ENDPOINT=${ENDPOINT:-localhost:9443}
DURATION_MIN=${DURATION_MIN:-60}

CURR_TIME=$(date +%s000) # in milliseconds
START_TIME=$(( CURR_TIME - DURATION_MIN * 60 * 1000 )) # in milliseconds

DASHBOARD_ID=${DASHBOARD_ID:-Q0AZXCdZk}

DASHBOARD_FETCH_URL="https://${ENDPOINT}/api/dashboards/uid/${DASHBOARD_ID}"

PANEL_LIST_JSON=$(curl -u admin:stackrox -sk "${DASHBOARD_FETCH_URL}" | jq '[.dashboard.panels[] | {id: .id, title: .title, queries: [.targets[].query]}]')

echo "${PANEL_LIST_JSON}" | jq -c '.[]' | while read -r panel_json; do
    panel_id=$(echo "${panel_json}" | jq .id)
    panel_title_file=$(echo "${panel_json}" | jq -r .title | sed -e "s/[^A-Za-z0-9._-]/_/g" | tr "[:upper:]" "[:lower:]")

    URL="https://${ENDPOINT}/render/d-solo/${DASHBOARD_ID}/core-dashboard?orgId=1&from=$START_TIME&to=$CURR_TIME&panelId=${panel_id}&width=2000&height=2000&tz=America%2FLos_Angeles"
	curl -u admin:stackrox -sk "${URL}" > "$DIR/captures/${panel_title_file}.png"
done

mkdir -p "$DIR/rawcaptures"

kubectl -n stackrox port-forward $(kubectl -n stackrox get po -l app=monitoring -o jsonpath={.items[].metadata.name}) 8086:8086 > /dev/null &
PID=$!
sleep 5

echo "${PANEL_LIST_JSON}" | jq -c '.[]' | while read -r panel_json; do
    queries=$(echo "${panel_json}" | jq .queries)
    panel_title_file=$(echo "${panel_json}" | jq -r .title | sed -e "s/[^A-Za-z0-9._-]/_/g" | tr "[:upper:]" "[:lower:]")

    count=0
    echo ${queries} | jq -rc '.[]' | while read -r query; do
        count=$(( count + 1))
        query=$(echo "${query}" | sed "s/\$timeFilter/time > now() - ${DURATION_MIN}m/g" | sed 's/\$__interval/1s/g')
        curl -s localhost:8086/query?db=telegraf_12h --data-urlencode "q=$query" > "$DIR/rawcaptures/${panel_title_file}_${count}.json"
    done
done

kill $PID

zip -jr "$DIR/rawmetrics.zip" "$DIR/rawcaptures"