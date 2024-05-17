import { allEnabled } from 'utils/featureFlagUtils';
import { BaseIntegrationDescriptor } from '../utils/integrationsList';

export function featureFlagDependencyFilter() {
    return (descriptor: BaseIntegrationDescriptor) => {
        const { featureFlagDependency } = descriptor;
        if (featureFlagDependency && featureFlagDependency.length > 0) {
            return allEnabled(featureFlagDependency);
        }
        return true;
    };
}

export function integrationTypeCounter(integrations: { type: string }[]) {
    return (type: string) =>
        integrations.filter((integration) => integration.type.toLowerCase() === type.toLowerCase())
            .length;
}
