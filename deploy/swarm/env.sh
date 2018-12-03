#!/usr/bin/env bash
set -e

export LOCAL_API_ENDPOINT="${LOCAL_API_ENDPOINT:-"localhost:8000"}"
echo "Local StackRox Central endpoint set to $LOCAL_API_ENDPOINT"

export CLUSTER_API_ENDPOINT="${CLUSTER_API_ENDPOINT:-central.prevent_net:443}"
echo "In-cluster Central endpoint set to $CLUSTER_API_ENDPOINT"

export ROX_HTPASSWD_AUTH=${ROX_HTPASSWD_AUTH:-true}
echo "ROX_HTPASSWD_AUTH set to $ROX_HTPASSWD_AUTH"
