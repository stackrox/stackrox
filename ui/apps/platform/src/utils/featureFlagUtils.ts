import type { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import type { FeatureFlagEnvVar } from 'types/featureFlag';

export type FeatureFlagPredicate = (isFeatureFlagEnabled: IsFeatureFlagEnabled) => boolean;

export function allEnabled(featureFlags: FeatureFlagEnvVar[]): FeatureFlagPredicate {
    return (isFeatureFlagEnabled: IsFeatureFlagEnabled): boolean => {
        return featureFlags.every((featureFlag) => isFeatureFlagEnabled(featureFlag));
    };
}
