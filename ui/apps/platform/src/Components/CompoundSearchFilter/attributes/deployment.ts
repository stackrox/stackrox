// If you're adding a new attribute, make sure to add it to "deploymentAttributes" as well

import type { CompoundSearchFilterAttribute } from '../types';

export const ID: CompoundSearchFilterAttribute = {
    displayName: 'ID',
    filterChipLabel: 'Deployment ID',
    searchTerm: 'Deployment ID',
    inputType: 'autocomplete',
};

export const Name: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Deployment name',
    searchTerm: 'Deployment',
    inputType: 'autocomplete',
};

export const Label: CompoundSearchFilterAttribute = {
    displayName: 'Label',
    filterChipLabel: 'Deployment label',
    searchTerm: 'Deployment Label',
    inputType: 'autocomplete',
};

export const Annotation: CompoundSearchFilterAttribute = {
    displayName: 'Annotation',
    filterChipLabel: 'Deployment annotation',
    searchTerm: 'Deployment Annotation',
    inputType: 'autocomplete',
};

export const ContainerType: CompoundSearchFilterAttribute = {
    displayName: 'Container type',
    filterChipLabel: 'Container type',
    searchTerm: 'Container Type',
    inputType: 'select',
    inputProps: {
        options: [
            { value: 'REGULAR', label: 'Regular' },
            { value: 'INIT', label: 'Init' },
        ],
    },
    featureFlagDependency: ['ROX_INIT_CONTAINER_SUPPORT'],
};

export const Inactive: CompoundSearchFilterAttribute = {
    displayName: 'Status',
    filterChipLabel: 'Deployment status',
    searchTerm: 'Inactive Deployment',
    inputType: 'select',
    inputProps: {
        options: [
            { value: 'false', label: 'Active' },
            { value: 'true', label: 'Inactive' },
        ],
    },
};

export const State: CompoundSearchFilterAttribute = {
    displayName: 'State',
    filterChipLabel: 'Deployment state',
    searchTerm: 'Deployment State',
    inputType: 'select',
    inputProps: {
        options: [
            { value: 'DEPLOYMENT_STATE_ACTIVE', label: 'Deployed' },
            { value: 'DEPLOYMENT_STATE_DELETED', label: 'Deleted' },
        ],
    },
    featureFlagDependency: ['ROX_DEPLOYMENT_SOFT_DELETION'],
};

export const deploymentAttributes = [Annotation, ContainerType, ID, Label, Name, Inactive, State];
