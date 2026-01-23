import { clusterNameAttribute as attributeForClusterName } from 'Components/CompoundSearchFilter/attributes/cluster';
import { Name as attributeForDeploymentName } from 'Components/CompoundSearchFilter/attributes/deployment';
import { Name as attributeForNamespaceName } from 'Components/CompoundSearchFilter/attributes/namespace';
import type {
    CompoundSearchFilterConfig,
    CompoundSearchFilterEntity,
} from 'Components/CompoundSearchFilter/types';

const entityForDeployment: CompoundSearchFilterEntity = {
    displayName: 'Deployment',
    searchCategory: 'DEPLOYMENTS',
    attributes: [attributeForDeploymentName],
};

const entityForNamespace: CompoundSearchFilterEntity = {
    displayName: 'Namespace',
    searchCategory: 'NAMESPACES',
    attributes: [attributeForNamespaceName],
};

const entityForCluster: CompoundSearchFilterEntity = {
    displayName: 'Cluster',
    searchCategory: 'CLUSTERS',
    attributes: [attributeForClusterName],
};

export const searchFilterConfig: CompoundSearchFilterConfig = [
    entityForCluster,
    entityForDeployment,
    entityForNamespace,
];
