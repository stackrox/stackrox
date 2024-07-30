// If you're adding a new attribute, make sure to add it to the "clusterAttributes" array as well

import { SearchFilterAttribute } from '../types';

const ID = {
    displayName: 'ID',
    filterChipLabel: 'Cluster ID',
    searchTerm: 'Cluster ID',
    inputType: 'autocomplete',
} as const;

const Name = {
    displayName: 'Name',
    filterChipLabel: 'Cluster name',
    searchTerm: 'Cluster',
    inputType: 'autocomplete',
} as const;

const Label = {
    displayName: 'Label',
    filterChipLabel: 'Cluster label',
    searchTerm: 'Cluster Label',
    inputType: 'autocomplete',
} as const;

const Type = {
    displayName: 'Type',
    filterChipLabel: 'Cluster type',
    searchTerm: 'Cluster Type',
    inputType: 'autocomplete',
} as const;

const PlatformType = {
    displayName: 'Platform Type',
    filterChipLabel: 'Platform type',
    searchTerm: 'Cluster Platform Type',
    inputType: 'autocomplete',
} as const;

export const clusterAttributes = [ID, Name, Label, Type, PlatformType] as const;

export type ClusterAttribute = (typeof clusterAttributes)[number]['displayName'];

export function getClusterAttributes(attributes?: ClusterAttribute[]): SearchFilterAttribute[] {
    if (!attributes || attributes.length === 0) {
        return clusterAttributes as unknown as SearchFilterAttribute[];
    }

    return clusterAttributes.filter((imageAttribute) => {
        return attributes.includes(imageAttribute.displayName);
    }) as SearchFilterAttribute[];
}
