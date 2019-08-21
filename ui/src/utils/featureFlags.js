export const types = {
    SHOW_DISALLOWED_CONNECTIONS: 'SHOW_DISALLOWED_CONNECTIONS'
};

// featureFlags defines UI specific feature flags.
const featureFlags = {
    [types.SHOW_DISALLOWED_CONNECTIONS]: false
};

// knownBackendFlags defines backend feature flags that are checked in the UI.
export const knownBackendFlags = {
    ROX_SENSOR_AUTOUPGRADE: 'ROX_SENSOR_AUTOUPGRADE',
    ROX_CONFIG_MGMT_UI: 'ROX_CONFIG_MGMT_UI',
    ROX_SCANNER_V2: 'ROX_SCANNER_V2'
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
