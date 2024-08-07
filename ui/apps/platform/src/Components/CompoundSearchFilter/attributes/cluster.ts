// If you're adding a new attribute, make sure to add it to "clusterAttributes" as well

import { CompoundSearchFilterAttribute } from '../types';

export const ID: CompoundSearchFilterAttribute = {
    displayName: 'ID',
    filterChipLabel: 'Cluster ID',
    searchTerm: 'Cluster ID',
    inputType: 'autocomplete',
};

export const Name: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Cluster name',
    searchTerm: 'Cluster',
    inputType: 'autocomplete',
};

export const Label: CompoundSearchFilterAttribute = {
    displayName: 'Label',
    filterChipLabel: 'Cluster label',
    searchTerm: 'Cluster Label',
    inputType: 'autocomplete',
};

export const Type: CompoundSearchFilterAttribute = {
    displayName: 'Type',
    filterChipLabel: 'Cluster type',
    searchTerm: 'Cluster Type',
    inputType: 'autocomplete',
};

export const PlatformType: CompoundSearchFilterAttribute = {
    displayName: 'Platform Type',
    filterChipLabel: 'Platform type',
    searchTerm: 'Cluster Platform Type',
    inputType: 'autocomplete',
};

export const clusterAttributes = [ID, Name, Label, Type, PlatformType];
