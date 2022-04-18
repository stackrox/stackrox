import { FeatureFlag } from 'types/featureFlagService.proto';

export const types = {
    SHOW_DISALLOWED_CONNECTIONS: 'SHOW_DISALLOWED_CONNECTIONS',
};

// featureFlags defines UI specific feature flags.
export const UIfeatureFlags = {
    [types.SHOW_DISALLOWED_CONNECTIONS]: false,
};

// knownBackendFlags defines backend feature flags that are checked in the UI.
export const knownBackendFlags = {
    ROX_ECR_AUTO_INTEGRATION: 'ROX_ECR_AUTO_INTEGRATION',
    ROX_POLICIES_PATTERNFLY: 'ROX_POLICIES_PATTERNFLY',
    ROX_SYSTEM_HEALTH_PF: 'ROX_SYSTEM_HEALTH_PF',
    ROX_NEW_POLICY_CATEGORIES: 'ROX_NEW_POLICY_CATEGORIES',
};

// isBackendFeatureFlagEnabled returns whether a feature flag retrieved from the backend is enabled.
// The default should never be required unless there's a programming error.
export const isBackendFeatureFlagEnabled = (
    backendFeatureFlags: FeatureFlag[],
    envVar: string,
    defaultVal = false
): boolean => {
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
