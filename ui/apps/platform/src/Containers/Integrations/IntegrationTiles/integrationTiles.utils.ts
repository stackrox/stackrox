import { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';

import { BaseIntegrationDescriptor } from '../utils/integrationsList';

export function featureFlagDependencyFilterer(isFeatureFlagEnabled: IsFeatureFlagEnabled) {
    return (descriptor: BaseIntegrationDescriptor) => {
        if (typeof descriptor.featureFlagDependency === 'string') {
            if (!isFeatureFlagEnabled(descriptor.featureFlagDependency)) {
                return false;
            }
        }
        return true;
    };
}

export function integrationTypeCounter(integrations: { type: string }[]) {
    return (type: string) =>
        integrations.filter((integration) => integration.type.toLowerCase() === type.toLowerCase())
            .length;
}
