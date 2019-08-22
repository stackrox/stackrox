#!/usr/bin/env bash

# Prints all resources of all types, optionally applying any label filters passed as arguments.

label_filter_select=""

for label_filter in "$@"; do
	if [[ "$label_filter" =~ ^(.*)=(.*)$ ]]; then
		key="${BASH_REMATCH[1]}"
		value="${BASH_REMATCH[2]}"
		label_filter_select="${label_filter_select} | select(.metadata.labels[\"${key}\"] == \"${value}\")"
	else
		echo >&2 "Filter argument must be of form key=value, got ${label_filter}"
		exit 1
	fi
done

kubectl api-resources -o name | egrep -v '^(events(\.events\.k8s\.io)?|componentstatuses|podmetrics|.*\.metrics\.k8s\.io)$' \
	| paste -sd, - | xargs kubectl -n stackrox get -o json 2>/dev/null \
	| jq -r '.items[] | select(
		  (.apiVersion == "v1" and .kind == "Secret" and (.type == "kubernetes.io/service-account-token" or .type == "kubernetes.io/dockerconfigjson" or .type == "kubernetes.io/dockercfg") | not) and
		  ((.metadata.ownerReferences // []) | length) == 0
		)'"${label_filter_select}"' | (.apiVersion + ":" + .kind + ":" + .metadata.name)'

