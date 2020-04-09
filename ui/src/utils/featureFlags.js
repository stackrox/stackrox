export const types = {
    SHOW_DISALLOWED_CONNECTIONS: 'SHOW_DISALLOWED_CONNECTIONS'
};

// featureFlags defines UI specific feature flags.
const featureFlags = {
    [types.SHOW_DISALLOWED_CONNECTIONS]: false
};

// knownBackendFlags defines backend feature flags that are checked in the UI.
export const knownBackendFlags = {
    ROX_VULN_MGMT_UI: 'ROX_VULN_MGMT_UI',
    ROX_ANALYST_NOTES_UI: 'ROX_ANALYST_NOTES_UI',
    ROX_EVENT_TIMELINE_UI: 'ROX_EVENT_TIMELINE_UI',
    ROX_TELEMETRY: 'ROX_TELEMETRY',
    ROX_DIAGNOSTIC_BUNDLE: 'ROX_DIAGNOSTIC_BUNDLE',
    ROX_REFRESH_TOKENS: 'ROX_REFRESH_TOKENS',
    ROX_ADMISSION_CONTROL_ENFORCE_ON_UPDATE: 'ROX_ADMISSION_CONTROL_ENFORCE_ON_UPDATE',
    ROX_DRY_RUN_JOB: 'ROX_DRY_RUN_JOB',
    ROX_BOOLEAN_POLICY_LOGIC: 'ROX_BOOLEAN_POLICY_LOGIC'
};

// isBackendFeatureFlagEnabled returns whether a feature flag retrieved from the backend is enabled.
// The default should never be required unless there's a programming error.
export const isBackendFeatureFlagEnabled = (backendFeatureFlags, envVar, defaultVal) => {
    const featureFlag = backendFeatureFlags.find(flag => flag.envVar === envVar);
    if (!featureFlag) {
        if (process.env.NODE_ENV === 'development') {
            throw new Error(`EnvVar ${envVar} not found in the backend list, possibly stale?`);
        }
        return defaultVal;
    }
    return featureFlag.enabled;
};

export default featureFlags;
