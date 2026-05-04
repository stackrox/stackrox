#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

# This file provides the function
#
#   deploy_stackrox_with_roxie_compat()
#
# which deploys StackRox using roxie, but allows configuration through environment variables for compatibility
# with existing tests. New tests should use deploy_stackrox_with_roxie() directly and provide configuration through
# override files instead of environment variables.
#
# The environment based configuration is implemented in the function roxie_override_from_environment_compat(),
# which translates environment variables into roxie overrides.

managed_by="stackrox-tests"

# Note: the caller must make sure to redirect stdin to /dev/null where needed.
retrying_kubectl() {
    "${TEST_ROOT}/scripts/retry-kubectl.sh" "$@"
}
export -f retrying_kubectl

# TODO: Make namespaces configurable?
# Implements a compatibility configuration layer.
# For new use-cases, please use deploy_stackrox_with_roxie() instead.
deploy_stackrox_with_roxie_compat() {
    local namespace="stackrox"
    info "Deploying StackRox with roxie (compat layer)"

    check_for_roxie

    if retrying_kubectl get ns "$namespace" </dev/null >/dev/null 2>&1; then
        # Deletes secrets created by previous invocation of this compat layer, e.g. default TLS secret for central.
        retrying_kubectl -n "$namespace" delete --ignore-not-found=true secrets -l app.kubernetes.io/managed-by="${managed_by}" --wait </dev/null
        retrying_kubectl -n "$namespace" delete --ignore-not-found=true configmap declarative-configurations --wait </dev/null
        retrying_kubectl -n "$namespace" delete --ignore-not-found=true configmap sensitive-declarative-configurations --wait </dev/null
    else
        retrying_kubectl create ns "${namespace}" </dev/null
    fi

    local override_file; override_file="$(mktemp)"

    # Prepare override file with some static settings coming from secured-cluster-cr.envsubst.yaml.
    merge_yaml "$override_file" <<EOF
securedCluster:
  spec:
    clusterName: remote
    processIndicators:
      excludeNamespaceRegex: namespace-without-persistence
EOF

    roxie_override_from_environment_compat "$override_file" "$namespace"

    deploy_stackrox_with_roxie "$namespace" "$override_file"
    rm -f "$override_file"
}

# This function translates environment settings into a roxie override.
# This is a compatibility function to allow using roxie with existing environment variable-based configuration for tests,
# without needing to switch to the new override file approach immediately.
roxie_override_from_environment_compat() {
    local override_file="$1"
    local namespace="$2"

    if [[ "${USE_KONFLUX_IMAGES:-false}" == "true" ]]; then
        info "Using Konflux-built downstream images for deployment."
        patch_yaml "$override_file" '.roxie.useKonfluxImages = true'
    fi

    handle_pod_security_policies

    info "Configuring TRUSTED_CA_FILE..."
    handle_trusted_ca_file "$override_file"

    info "Configuring TLS..."
    handle_default_tls_settings "$override_file" "$namespace"

    info "Configuring load balancer..."
    handle_load_balancer_setting "$override_file"

    info "Configuring custom central environment..."
    {
        # These are set in env.sh and picked up roxctl/central/generate/generate.go:updateConfig().
        env_with_default ROX_DEVELOPMENT_BUILD "true"   # pkg/devbuild/setting.go.
        env_with_default ROX_HOTRELOAD "false"          # pkg/env/hot_reload.go.
        env_with_default ROX_NETWORK_ACCESS_LOG "false" # pkg/env/networklog.go.

        # Set in tests/e2e/lib.sh:export_test_environment().
        env_with_default ROX_DECLARATIVE_CONFIGURATION "true" # pkg/env/declarative_config.go.

        # Set in deploy_central_via_operator().
        env_with_default ROX_PROCESSES_LISTENING_ON_PORT "true" # pkg/env/processes_listening_on_port.go.
        env_with_default ROX_RISK_REPROCESSING_INTERVAL "15s"   # pkg/env/reprocessing_interval.go.

        # Set in tests/e2e/lib.sh:export_test_environment() and deploy_central_via_operator():
        env_with_default ROX_REGISTRY_RESPONSE_TIMEOUT "90s"          # pkg/env/registry.go.
        env_with_default ROX_REGISTRY_CLIENT_TIMEOUT "120s"           # pkg/env/registry.go.
        env_with_default ROX_BASELINE_GENERATION_DURATION "1m"        # pkg/env/baseline.go.
        env_with_default ROX_NETWORK_BASELINE_OBSERVATION_PERIOD "2m" # pkg/env/network_baseline_observation_period.go.
        env_with_default ROX_TELEMETRY_STORAGE_KEY_V1 "DISABLED"      # pkg/env/telemetry.go.

        # Configured by test suites, picked up by roxctl/central/generate/generate.go:updateConfig().
        env_with_default ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL # pkg/env/sensor.go.

        # Set in export_test_environment().
        env_with_default ROX_NETFLOW_BATCHING "true"       # pkg/env/sensor.go.
        env_with_default ROX_NETFLOW_CACHE_LIMITING "true" # pkg/env/sensor.go.

        # Not associated with default values.
        env_with_default ROX_TELEMETRY_API_WHITELIST # central/telemetry/centralclient/client.go
        env_with_default ROX_TELEMETRY_ENDPOINT      # pkg/env/telemetry.go

        collect_feature_flags

        # CGO checks. Set in deploy_central_via_operator().
        if [[ "${CGO_CHECKS:-}" == "true" ]]; then
            env_with_default GOEXPERIMENT "cgocheck2"
            env_with_default MUTEX_WATCHDOG_TIMEOUT_SECS "15"
        fi
    } | while read -r var_val; do
        local name="${var_val%%=*}"
        local value="${var_val#*=}"
        info "  ${name}=${value}"
        set_custom_env "$override_file" "central" "$name" "$value"
        ci_export "$name" "$value"
    done

    info "Configuring custom securedCluster environment..."
    {
        # Set in export_test_environment() and deploy_sensor_via_operator().
        env_with_default ROX_NETFLOW_BATCHING "true"       # pkg/env/sensor.go.
        env_with_default ROX_NETFLOW_CACHE_LIMITING "true" # pkg/env/sensor.go.

        collect_feature_flags

    } | while read -r var_val; do
        local name="${var_val%%=*}"
        local value="${var_val#*=}"
        info "  ${name}=${value}"
        set_custom_env "$override_file" "securedCluster" "$name" "$value"
        ci_export "$name" "$value"
    done

    info "Configuring custom securedCluster/collector environment..."
    {
        # Set in deploy_sensor_via_operator() and deploy_sensor().
        env_with_default ROX_AFTERGLOW_PERIOD "15"
        env_with_default ROX_COLLECTOR_INTROSPECTION_ENABLE "true"
        # Set in export_test_environment().
        if [[ "${CI_JOB_NAME:-}" =~ gke ]]; then
            # GKE uses this network for services. Consider it as a private subnet.
            env_with_default ROX_NON_AGGREGATED_NETWORKS "34.118.224.0/20"
        else
            env_with_default ROX_NON_AGGREGATED_NETWORKS
        fi
    } | while read -r var_val; do
        local name="${var_val%%=*}"
        local value="${var_val#*=}"
        info "  ${name}=${value} for DaemonSet/collector"
        set_overlay_env "$override_file" "securedCluster" "apps/v1" "DaemonSet" "collector" "collector" "$name" "$value"
        ci_export "$name" "$value"
    done

    info "Configuring scanner V4..."
    handle_scanner_v4_setting "$override_file" ".central.spec.scannerV4.scannerComponent"
    handle_scanner_v4_setting "$override_file" ".securedCluster.spec.scannerV4.scannerComponent"

    info "Configuring declarative configuration..."
    handle_declarative_configuration "$override_file"

    info "Configuring file activity monitoring mode..."
    handle_file_activity_monitoring "$override_file"
}

# Emit feature flags, enabling injection into roxie overrides, rendering them overwritable using
# environment variables.
collect_feature_flags() {
    # Disabled by default in StackRox, but enabled by default for test deployments.
    env_with_default ROX_CISA_KEV "true"
    env_with_default ROX_VULNERABILITY_REPORTS_ENHANCED_FILTERING "true"
    env_with_default ROX_NODE_VULNERABILITY_REPORTS "true"
    env_with_default ROX_TAILORED_PROFILES "true"
    env_with_default ROX_INIT_CONTAINER_SUPPORT "true"
    env_with_default ROX_LABEL_BASED_POLICY_SCOPING "true"
    env_with_default ROX_POLICY_CRITERIA_MODAL "true"
    env_with_default ROX_VULN_MGMT_LEGACY_SNOOZE "true"
    env_with_default ROX_NETWORK_GRAPH_AGGREGATE_EXT_IPS "true"

    # Enabled by default in StackRox, but disabled by default for test deployments.
    env_with_default ROX_NETWORK_GRAPH_EXTERNAL_IPS "false"

    # Defaults unchanged, but can be modified by test suites.
    env_with_default ROX_SENSITIVE_FILE_ACTIVITY
    env_with_default ROX_BASE_IMAGE_DETECTION
}

handle_pod_security_policies() {
    local pod_security_policies="${POD_SECURITY_POLICIES:-false}"
    if [[ "$pod_security_policies" == "true" ]]; then
        info "WARNING: roxie-based deployments do not support PodSecurityPolicies"
    fi
    export POD_SECURITY_POLICIES="false"
    ci_export POD_SECURITY_POLICIES "$POD_SECURITY_POLICIES"
}

handle_scanner_v4_setting() {
    local override_file="$1"
    local path="$2"
    local rox_scanner_v4="${ROX_SCANNER_V4:-false}" # To match the previous defaulting

    case "$rox_scanner_v4" in
        true)
            patch_yaml "$override_file" "${path} = \"Enabled\""
            ;;
        false)
            patch_yaml "$override_file" "${path} = \"Disabled\""
            ;;
        *)
            die "Unsupported value for ROX_SCANNER_V4: $rox_scanner_v4"
            ;;
    esac
}

handle_trusted_ca_file() {
    local override_file="$1"
    local trusted_ca_file="${TRUSTED_CA_FILE:-}"

    if [[ -n "$trusted_ca_file" ]]; then
        [[ -f "$trusted_ca_file" ]] || die "Trusted CA file not found: $trusted_ca_file"
        trusted_ca_as_string=$(jq -Rs . < "$trusted_ca_file")
        merge_yaml "$override_file" <<EOF
central:
  spec:
    tls:
      additionalCAs:
      - name: additional-ca
        content: $trusted_ca_as_string
EOF
    fi
}

handle_default_tls_settings() {
    local override_file="$1"
    local namespace="$2"
    local rox_default_tls_key_file="${ROX_DEFAULT_TLS_KEY_FILE:-}"
    local rox_default_tls_cert_file="${ROX_DEFAULT_TLS_CERT_FILE:-}"

    if [[ -n "$rox_default_tls_key_file" && -n "$rox_default_tls_cert_file" ]]; then
        info "Setting up default TLS certificate"
        [[ -f "$rox_default_tls_key_file" ]] || die "TLS key file not found: $rox_default_tls_key_file"
        [[ -f "$rox_default_tls_cert_file" ]] || die "TLS cert file not found: $rox_default_tls_cert_file"
        info "Creating TLS secret with test certificates"
        local central_default_tls_secret_name="central-default-tls-secret"
        retrying_kubectl -n "${namespace}" apply -f - << EOF
apiVersion: v1
kind: Secret
type: kubernetes.io/tls
metadata:
  name: "${central_default_tls_secret_name}"
  labels:
    app.kubernetes.io/managed-by: "${managed_by}"
data:
  tls.key: $(base64 < "$rox_default_tls_key_file" | tr -d '\n')
  tls.crt: $(base64 < "$rox_default_tls_cert_file" | tr -d '\n')
EOF
        merge_yaml "$override_file" << EOF
central:
  spec:
    central:
      defaultTLSSecret:
        name: "${central_default_tls_secret_name}"
EOF
    fi
}

handle_load_balancer_setting() {
    local override_file="$1"
    local load_balancer="${LOAD_BALANCER:-}"

    case "$load_balancer" in
    "")
        ;;
    lb)
        patch_yaml "$override_file" ".central.spec.central.exposure.loadBalancer.enabled = true"
        ;;
    route)
        patch_yaml "$override_file" ".central.spec.central.exposure.route.enabled = true"
        ;;
    *)
        die "Unsupported value for LOAD_BALANCER: $load_balancer"
        ;;
    esac
}

handle_declarative_configuration() {
    local override_file="$1"
    merge_yaml "$override_file" <<EOF
central:
  spec:
    central:
      declarativeConfiguration:
        configMaps:
        - name: "declarative-configurations"
        secrets:
        - name: "sensitive-declarative-configurations"
EOF
}

handle_file_activity_monitoring() {
    local override_file="$1"
    local sfa_agent="${SFA_AGENT:-false}"

    case "$sfa_agent" in
    true)
        patch_yaml "$override_file" '.securedCluster.spec.perNode.fileActivityMonitoring.mode = "Enabled"'
        ;;
    false)
        patch_yaml "$override_file" '.securedCluster.spec.perNode.fileActivityMonitoring.mode = "Disabled"'
        ;;
    *)
        die "Unsupported value for SFA_AGENT: ${sfa_agent}"
        ;;
    esac
}

env_with_default() {
    local name="$1"
    local default_value="${2:-}"
    local value="${!name:-}"
    if [[ -n "$value" ]]; then
        echo "${name}=${value}"
    elif [[ -n "$default_value" ]]; then
        echo "${name}=${default_value}"
    fi
}
