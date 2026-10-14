import {
    Name as attributeForDeploymentName,
    deploymentAttributes,
} from 'Components/CompoundSearchFilter/attributes/deployment';
import type {
    CompoundSearchFilterConfig,
    CompoundSearchFilterEntity,
} from 'Components/CompoundSearchFilter/types';

// The Network Graph manages Cluster, Namespace, and Deployment selection through dedicated
// scope selectors, which write those values into the same URL search filter. Exclude the
// Deployment name attribute so the compound search filter does not compete with the Deployment
// scope selector for the "Deployment" search key (matches the legacy searchOptionExclusions).
const entityForDeployment: CompoundSearchFilterEntity = {
    displayName: 'Deployment',
    searchCategory: 'DEPLOYMENTS',
    attributes: deploymentAttributes.filter(
        (attribute) => attribute !== attributeForDeploymentName
    ),
};

export const searchFilterConfig: CompoundSearchFilterConfig = [entityForDeployment];
