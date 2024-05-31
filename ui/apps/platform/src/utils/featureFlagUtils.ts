import { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import { FeatureFlagEnvVar } from 'types/featureFlag';

// Given an array of feature flags, higher-order functions return true or false based on
// whether all feature flags are enabled or disabled

export type FeatureFlagPredicate = (isFeatureFlagEnabled: IsFeatureFlagEnabled) => boolean;

export function allEnabled(featureFlags: FeatureFlagEnvVar[]): FeatureFlagPredicate {
    return (isFeatureFlagEnabled: IsFeatureFlagEnabled): boolean => {
        return featureFlags.every((featureFlag) => isFeatureFlagEnabled(featureFlag));
    };
}

export function allDisabled(featureFlags: FeatureFlagEnvVar[]): FeatureFlagPredicate {
    return (isFeatureFlagEnabled: IsFeatureFlagEnabled): boolean => {
        return featureFlags.every((featureFlag) => !isFeatureFlagEnabled(featureFlag));
    };
}
