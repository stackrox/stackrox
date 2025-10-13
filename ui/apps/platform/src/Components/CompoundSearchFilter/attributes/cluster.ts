import type { CompoundSearchFilterAttribute } from '../types';

export const clusterIdAttribute: CompoundSearchFilterAttribute = {
    displayName: 'ID',
    filterChipLabel: 'Cluster ID',
    searchTerm: 'Cluster ID',
    inputType: 'autocomplete',
};

export const clusterNameAttribute: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Cluster name',
    searchTerm: 'Cluster',
    inputType: 'autocomplete',
};

export const clusterLabelAttribute: CompoundSearchFilterAttribute = {
    displayName: 'Label',
    filterChipLabel: 'Cluster label',
    searchTerm: 'Cluster Label',
    inputType: 'autocomplete',
};

export const clusterTypeAttribute: CompoundSearchFilterAttribute = {
    displayName: 'Type',
    filterChipLabel: 'Cluster type',
    searchTerm: 'Cluster Type',
    inputType: 'autocomplete',
};

export const clusterPlatformTypeAttribute: CompoundSearchFilterAttribute = {
    displayName: 'Platform type',
    filterChipLabel: 'Platform type',
    searchTerm: 'Cluster Platform Type',
    inputType: 'autocomplete',
};

export const clusterKubernetesVersionAttribute: CompoundSearchFilterAttribute = {
    displayName: 'Kubernetes version',
    filterChipLabel: 'Cluster kubernetes version',
    searchTerm: 'Cluster Kubernetes Version',
    inputType: 'autocomplete',
};
