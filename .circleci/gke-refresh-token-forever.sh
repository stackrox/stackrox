#!/usr/bin/env bash

REAL_KUBECONFIG="${KUBECONFIG:-${HOME}/.kube/config}"

# refresh token every 15m
while sleep 900; do
	gcloud config config-helper --force-auth-refresh >/dev/null
	echo >/tmp/kubeconfig-new
	chmod 0600 /tmp/kubeconfig-new
	KUBECONFIG=/tmp/kubeconfig-new gcloud container clusters get-credentials --project stackrox-ci --zone "$ZONE" "$CLUSTER_NAME"
	KUBECONFIG=/tmp/kubeconfig-new kubectl get ns >/dev/null
	mv /tmp/kubeconfig-new "$REAL_KUBECONFIG"
done
