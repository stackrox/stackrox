export const types = {
    SHOW_DISALLOWED_CONNECTIONS: 'SHOW_DISALLOWED_CONNECTIONS',
};

// featureFlags defines UI specific feature flags.
const featureFlags = {
    [types.SHOW_DISALLOWED_CONNECTIONS]: false,
};

// knownBackendFlags defines backend feature flags that are checked in the UI.
export const knownBackendFlags = {
    ROX_ANALYST_NOTES_UI: 'ROX_ANALYST_NOTES_UI',
    ROX_EVENT_TIMELINE_CLUSTERED_EVENTS_UI: 'ROX_EVENT_TIMELINE_CLUSTERED_EVENTS_UI',
    ROX_POLICY_IMPORT_EXPORT: 'ROX_POLICY_IMPORT_EXPORT',
    ROX_ADMISSION_CONTROL_ENFORCE_ON_UPDATE: 'ROX_ADMISSION_CONTROL_ENFORCE_ON_UPDATE',
    ROX_AUTH_TEST_MODE_UI: 'ROX_AUTH_TEST_MODE_UI',
    ROX_CURRENT_USER_INFO: 'ROX_CURRENT_USER_INFO',
    ROX_SUPPORT_SLIM_COLLECTOR_MODE: 'ROX_SUPPORT_SLIM_COLLECTOR_MODE',
    ROX_AWS_SECURITY_HUB_INTEGRATION: 'ROX_AWS_SECURITY_HUB_INTEGRATION',
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

export default featureFlags;
