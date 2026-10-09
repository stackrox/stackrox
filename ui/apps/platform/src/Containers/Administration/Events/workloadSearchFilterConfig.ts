import { clusterNameAttribute } from 'Components/CompoundSearchFilter/attributes/cluster';
import { Name as DeploymentName } from 'Components/CompoundSearchFilter/attributes/deployment';
import { Name as NamespaceName } from 'Components/CompoundSearchFilter/attributes/namespace';
import type { CompoundSearchFilterConfig } from 'Components/CompoundSearchFilter/types';

export const workloadSearchFilterConfig: CompoundSearchFilterConfig = [
    {
        displayName: 'Cluster',
        searchCategory: 'CLUSTERS',
        attributes: [clusterNameAttribute],
    },
    {
        displayName: 'Namespace',
        searchCategory: 'NAMESPACES',
        attributes: [NamespaceName],
    },
    {
        displayName: 'Deployment',
        searchCategory: 'DEPLOYMENTS',
        attributes: [DeploymentName],
    },
];
