#!/usr/bin/env bash
set -euo pipefail

logmein() {
  target_url="$(curl -sSkf -u "admin:${STACKROX_ADMIN_PASSWORD}" -w '%{redirect_url}' "https://localhost:8000/sso/providers/basic/4df1b98c-24ed-4073-a9ad-356aec6bb62d/challenge?micro_ts=0")"
  if [[ -z "${target_url}" ]]; then
    >&2 echo "UNEXPECTED: Could not determine target URL"
    return
  fi

  auth_error="$(sed -E '/^.*#error=([^&]+)(&.*)?$/!d;s//\1/' <<<"$target_url" | tr '+' ' ')"
  if [[ -n "$auth_error" ]]; then
    >&2 echo "Authentication error: ${auth_error}"
  fi

  echo "Logging you in via ${target_url} ..."
  platform="$(uname)"
  if [[ "$platform" == "Linux" ]]; then
      xdg-open "${target_url}" >/dev/null &
  elif [[ "$platform" == "Darwin" ]]; then
      open "${target_url}" &
  else
      >&2 echo "Unsupported platform '$platform', please open ${target_url} in a browser"
  fi
}

SMALL_INSTALL=false

case "${1:-}" in
    -h|--help)
        echo -e "Usage:\n\tquick-helm-install.sh [options]"
        echo " "
        echo "Installs StackRox via Helm charts."
        echo " "
        echo "options:"
        echo "-h, --help                show brief help"
        echo "-s, --small               reduce StackRox resource requirements for small clusters"
        exit 0
        ;;
    -s|--small)
        SMALL_INSTALL=true
        ;;
esac

echo "Adding the stackrox/helm-charts/opensource repository to Helm."

helm repo add stackrox https://raw.githubusercontent.com/stackrox/helm-charts/main/opensource/

echo "Generating STACKROX_ADMIN_PASSWORD"

STACKROX_ADMIN_PASSWORD="$(openssl rand -base64 20 | tr -d '/=+')"

echo "Installing stackrox-central-services"

installflags=()
if [[ "$SMALL_INSTALL" == "true" ]]; then
    installflags+=('--set' 'central.resources.requests.memory=1Gi')
    installflags+=('--set' 'central.resources.requests.cpu=1')
    installflags+=('--set' 'central.resources.limits.memory=4Gi')
    installflags+=('--set' 'central.resources.limits.cpu=1')
    installflags+=('--set' 'central.db.resources.requests.memory=1Gi')
    installflags+=('--set' 'central.db.resources.requests.cpu=500m')
    installflags+=('--set' 'central.db.resources.limits.memory=4Gi')
    installflags+=('--set' 'central.db.resources.limits.cpu=1')
    installflags+=('--set' 'scanner.autoscaling.disable=true')
    installflags+=('--set' 'scanner.replicas=1')
    installflags+=('--set' 'scanner.resources.requests.memory=500Mi')
    installflags+=('--set' 'scanner.resources.requests.cpu=500m')
    installflags+=('--set' 'scanner.resources.limits.memory=2500Mi')
    installflags+=('--set' 'scanner.resources.limits.cpu=2000m')
fi

helm install -n stackrox --create-namespace stackrox-central-services stackrox/stackrox-central-services \
 --set central.adminPassword.value="${STACKROX_ADMIN_PASSWORD}" \
 --set central.persistence.none=true \
 "${installflags[@]+"${installflags[@]}"}"

kubectl -n stackrox rollout status deploy/central --timeout=3m

echo "Setting up central port-forward"

kubectl -n stackrox port-forward deploy/central --pod-running-timeout=1m0s 8000:8443 > /dev/null 2>&1 &

echo "Generating an init bundle with stackrox-secured-cluster-services provisioning secrets"

kubectl -n stackrox exec deploy/central -- roxctl --insecure-skip-tls-verify \
  --password "${STACKROX_ADMIN_PASSWORD}" \
  central init-bundles generate stackrox-init-bundle --output - 1> stackrox-init-bundle.yaml

installflags=()
if [[ "$SMALL_INSTALL" == "true" ]]; then
    installflags+=('--set' 'sensor.resources.requests.memory=500Mi')
    installflags+=('--set' 'sensor.resources.requests.cpu=500m')
    installflags+=('--set' 'sensor.resources.limits.memory=500Mi')
    installflags+=('--set' 'sensor.resources.limits.cpu=500m')
fi

echo "Installing stackrox-secured-cluster-services"

helm install -n stackrox stackrox-secured-cluster-services stackrox/stackrox-secured-cluster-services \
 -f stackrox-init-bundle.yaml --set clusterName="my-secured-cluster" \
 "${installflags[@]+"${installflags[@]}"}"

echo "Logging into StackRox in the browser"

logmein

echo -e "
\033[1;31mStackRox is now installed!\033[0m

You may access the dashboard via https://localhost:8000/main/dashboard, the user is admin.

Consult these documents for additional information on customizing your Helm installation:
https://docs.openshift.com/acs/installing/installing_other/install-central-other.html#install-using-helm-customizations-other
https://docs.openshift.com/acs/installing/installing_other/install-secured-cluster-other.html#configure-secured-cluster-services-helm-chart-customizations-other

STACKROX_ADMIN_PASSWORD='$STACKROX_ADMIN_PASSWORD'
Above is your automatically generated stackrox admin password. Please store it securely, as you will need it during further configuration.
In your current directory an init bundle \"stackrox-init-bundle.yaml\" was created, store it safely in case you are planning to provision more secured clusters with it, or delete it otherwise."

