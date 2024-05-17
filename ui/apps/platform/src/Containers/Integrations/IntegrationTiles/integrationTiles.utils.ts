import { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';

import { BaseIntegrationDescriptor } from '../utils/integrationsList';

export function featureFlagDependencyFilterer(isFeatureFlagEnabled: IsFeatureFlagEnabled) {
    return (descriptor: BaseIntegrationDescriptor) => {
        const { featureFlagDependency } = descriptor;
        if (featureFlagDependency && featureFlagDependency.length > 0) {
            featureFlagDependency.forEach((featureFlag) => {
                if (!isFeatureFlagEnabled(featureFlag)) {
                    return false;
                }
            });
        }
        return true;
    };
}

export function integrationTypeCounter(integrations: { type: string }[]) {
    return (type: string) =>
        integrations.filter((integration) => integration.type.toLowerCase() === type.toLowerCase())
            .length;
}
