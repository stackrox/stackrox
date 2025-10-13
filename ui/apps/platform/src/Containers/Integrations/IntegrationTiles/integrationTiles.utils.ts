import type { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import { allEnabled } from 'utils/featureFlagUtils';

import type { BaseIntegrationDescriptor } from '../utils/integrationsList';

export function featureFlagDependencyFilterer(isFeatureFlagEnabled: IsFeatureFlagEnabled) {
    return (descriptor: BaseIntegrationDescriptor) => {
        const { featureFlagDependency } = descriptor;
        if (featureFlagDependency && featureFlagDependency.length > 0) {
            return allEnabled(featureFlagDependency)(isFeatureFlagEnabled);
        }
        return true;
    };
}

export function integrationTypeCounter(integrations: { type: string }[]) {
    return (type: string) =>
        integrations.filter((integration) => integration.type.toLowerCase() === type.toLowerCase())
            .length;
}
