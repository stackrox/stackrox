#!/usr/bin/env bash
set -eou pipefail

while true; do
	kubectl -n stackrox top pod
	sleep 1
done
