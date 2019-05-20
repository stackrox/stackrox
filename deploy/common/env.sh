#!/usr/bin/env bash
set -e

export CLUSTER_API_ENDPOINT="${CLUSTER_API_ENDPOINT:-central.stackrox:443}"
echo "In-cluster Central endpoint set to $CLUSTER_API_ENDPOINT"

export RUNTIME_SUPPORT=${RUNTIME_SUPPORT:-ebpf}
echo "RUNTIME_SUPPORT set to $RUNTIME_SUPPORT"

export SCANNER_SUPPORT=${SCANNER_SUPPORT:-false}
echo "SCANNER_SUPPORT set to $SCANNER_SUPPORT"

export OFFLINE_MODE=${OFFLINE_MODE:-false}
echo "OFFLINE_MODE set to $OFFLINE_MODE"

export ROX_HTPASSWD_AUTH=${ROX_HTPASSWD_AUTH:-true}
echo "ROX_HTPASSWD_AUTH set to $ROX_HTPASSWD_AUTH"

export MONITORING_SUPPORT=${MONITORING_SUPPORT:-true}
echo "MONITORING_SUPPORT set to ${MONITORING_SUPPORT}"

export CLUSTER=${CLUSTER:-remote}
echo "CLUSTER set to $CLUSTER"

export STORAGE="${STORAGE:-none}"
echo "STORAGE set to ${STORAGE}"

export OUTPUT_FORMAT="${OUTPUT_FORMAT:-kubectl}"
echo "OUTPUT_FORMAT set to ${OUTPUT_FORMAT}"

export LOAD_BALANCER="${LOAD_BALANCER:-none}"
echo "LOAD_BALANCER set to ${LOAD_BALANCER}"

export MONITORING_LOAD_BALANCER="${MONITORING_LOAD_BALANCER:-none}"
echo "MONITORING_LOAD_BALANCER set to ${MONITORING_LOAD_BALANCER}"

export ADMISSION_CONTROLLER="${ADMISSION_CONTROLLER:-false}"
echo "ADMISSION_CONTROLLER set to ${ADMISSION_CONTROLLER}"

export ROX_NETWORK_POLICY_GENERATOR=true
echo "ROX_NETWORK_POLICY_GENERATOR is set to ${ROX_NETWORK_POLICY_GENERATOR}"

export ROX_K8S_RBAC=${ROX_K8S_RBAC:-false}
echo "ROX_K8S_RBAC is set to ${ROX_K8S_RBAC}"

export ROX_PROCESS_WHITELIST=${ROX_PROCESS_WHITELIST:-false}
echo "ROX_PROCESS_WHITELIST is set to ${ROX_PROCESS_WHITELIST}"

export ROX_DEVELOPMENT_BUILD=true
echo "ROX_DEVELOPMENT_BUILD is set to ${ROX_DEVELOPMENT_BUILD}"

export API_ENDPOINT="${API_ENDPOINT:-localhost:8000}"
echo "API_ENDPOINT is set to ${API_ENDPOINT}"

export TRUSTED_CA_FILE="${TRUSTED_CA_FILE:-}"
if [[ -n "${TRUSTED_CA_FILE}" ]]; then
  [[ -f "${TRUSTED_CA_FILE}" ]] || { echo "Trusted CA file ${TRUSTED_CA_FILE} not found"; return 1; }
  echo "TRUSTED_CA_FILE is set to ${TRUSTED_CA_FILE}"
else
  echo "No TRUSTED_CA_FILE provided"
fi

export ROX_DEFAULT_TLS_CERT_FILE="${ROX_DEFAULT_TLS_CERT_FILE:-}"
export ROX_DEFAULT_TLS_KEY_FILE="${ROX_DEFAULT_TLS_KEY_FILE:-}"

if [[ -n "$ROX_DEFAULT_TLS_CERT_FILE" ]]; then
	[[ -f "$ROX_DEFAULT_TLS_CERT_FILE" ]] || { echo "Default TLS certificate ${ROX_DEFAULT_TLS_CERT_FILE} not found"; return 1; }
	[[ -f "$ROX_DEFAULT_TLS_KEY_FILE" ]] || { echo "Default TLS key ${ROX_DEFAULT_TLS_KEY_FILE} not found"; return 1; }
	echo "Using default TLS certificate/key material from $ROX_DEFAULT_TLS_CERT_FILE, $ROX_DEFAULT_TLS_KEY_FILE"
elif [[ -n "$ROX_DEFAULT_TLS_KEY_FILE" ]]; then
	echo "ROX_DEFAULT_TLS_KEY_FILE is nonempty, but ROX_DEFAULT_TLS_CERT_FILE is"
	return 1
else
	echo "No default TLS certificates provided"
fi
