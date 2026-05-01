import {
    clusterIdAttribute,
    clusterLabelAttribute,
    clusterNameAttribute,
} from 'Components/CompoundSearchFilter/attributes/cluster';
import {
    Annotation as deploymentAnnotationAttribute,
    ID as deploymentIdAttribute,
    Label as deploymentLabelAttribute,
    Name as deploymentNameAttribute,
} from 'Components/CompoundSearchFilter/attributes/deployment';
import {
    Annotation as namespaceAnnotationAttribute,
    ID as namespaceIdAttribute,
    Label as namespaceLabelAttribute,
    Name as namespaceNameAttribute,
} from 'Components/CompoundSearchFilter/attributes/namespace';
import type { CompoundSearchFilterEntity } from 'Components/CompoundSearchFilter/types';

const searchFilterEntityForCluster: CompoundSearchFilterEntity = {
    displayName: 'Cluster',
    searchCategory: 'CLUSTERS',
    attributes: [
        // 'Cluster Annotation' is not a search filter
        clusterIdAttribute,
        clusterLabelAttribute,
        clusterNameAttribute,
    ],
};

const searchFilterEntityForDeployment: CompoundSearchFilterEntity = {
    displayName: 'Deployment',
    searchCategory: 'DEPLOYMENTS',
    attributes: [
        deploymentAnnotationAttribute,
        deploymentIdAttribute,
        deploymentLabelAttribute,
        deploymentNameAttribute,
    ],
};

const searchFilterEntityForNamespace: CompoundSearchFilterEntity = {
    displayName: 'Namespace',
    searchCategory: 'NAMESPACES',
    attributes: [
        namespaceAnnotationAttribute,
        namespaceIdAttribute,
        namespaceLabelAttribute,
        namespaceNameAttribute,
    ],
};

// Node vulnerability report configuration.
export const searchFilterConfigForCluster = [searchFilterEntityForCluster];

// Virtual machine vulnerability report configuration.
export const searchFilterConfigForClusterNamespace = [
    searchFilterEntityForCluster,
    searchFilterEntityForNamespace,
];

// Image vulnerability and violation report configuration.
export const searchFilterConfigForClusterNamespaceDeployment = [
    searchFilterEntityForCluster,
    searchFilterEntityForDeployment,
    searchFilterEntityForNamespace,
];
