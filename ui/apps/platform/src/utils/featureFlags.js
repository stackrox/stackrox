export const types = {
    SHOW_DISALLOWED_CONNECTIONS: 'SHOW_DISALLOWED_CONNECTIONS',
};

// featureFlags defines UI specific feature flags.
export const UIfeatureFlags = {
    [types.SHOW_DISALLOWED_CONNECTIONS]: false,
};

// knownBackendFlags defines backend feature flags that are checked in the UI.
export const knownBackendFlags = {
    ROX_SUPPORT_SLIM_COLLECTOR_MODE: 'ROX_SUPPORT_SLIM_COLLECTOR_MODE',
    ROX_AWS_SECURITY_HUB_INTEGRATION: 'ROX_AWS_SECURITY_HUB_INTEGRATION',
    ROX_NETWORK_GRAPH_PORTS: 'ROX_NETWORK_GRAPH_PORTS',
    ROX_NETWORK_FLOWS_SEARCH_FILTER_UI: 'ROX_NETWORK_FLOWS_SEARCH_FILTER_UI',
    ROX_NETWORK_GRAPH_EXTERNAL_SRCS: 'ROX_NETWORK_GRAPH_EXTERNAL_SRCS',
    ROX_SYSLOG_INTEGRATION: 'ROX_SYSLOG_INTEGRATION',
    ROX_NETWORK_DETECTION: 'ROX_NETWORK_DETECTION',
    ROX_NETWORK_DETECTION_BASELINE_VIOLATION: 'ROX_NETWORK_DETECTION_BASELINE_VIOLATION',
    ROX_SENSOR_INSTALLATION_EXPERIENCE: 'ROX_SENSOR_INSTALLATION_EXPERIENCE',
    ROX_HOST_SCANNING: 'ROX_HOST_SCANNING',
    ROX_K8S_EVENTS_DETECTION: 'ROX_K8S_EVENTS_DETECTION',
};

// isBackendFeatureFlagEnabled returns whether a feature flag retrieved from the backend is enabled.
// The default should never be required unless there's a programming error.
export const isBackendFeatureFlagEnabled = (backendFeatureFlags, envVar, defaultVal) => {
    const featureFlag = backendFeatureFlags.find((flag) => flag.envVar === envVar);
    if (!featureFlag) {
        if (process.env.NODE_ENV === 'development') {
            // eslint-disable-next-line no-console
            console.warn(`EnvVar ${envVar} not found in the backend list, possibly stale?`);
        }
        return defaultVal;
    }
    return featureFlag.enabled;
};
