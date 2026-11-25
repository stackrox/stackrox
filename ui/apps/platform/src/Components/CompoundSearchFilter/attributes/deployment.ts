// New attributes go in `deploymentAttributes` too, so they render wherever the Deployment entity is
// shown. Exception: page-scoped fields (Type, ServiceAccount, Secret) stay out of the aggregate to
// avoid surfacing elsewhere, but remain exported so searchUtils can register them for regex.

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

// Page-scoped fields (see note at top): kept out of deploymentAttributes on purpose.
export const Type: CompoundSearchFilterAttribute = {
    displayName: 'Type',
    filterChipLabel: 'Deployment type',
    searchTerm: 'Deployment Type',
    inputType: 'autocomplete',
};

export const ServiceAccount: CompoundSearchFilterAttribute = {
    displayName: 'Service account',
    filterChipLabel: 'Service account',
    searchTerm: 'Service Account',
    inputType: 'autocomplete',
};

export const Secret: CompoundSearchFilterAttribute = {
    displayName: 'Secret',
    filterChipLabel: 'Secret',
    searchTerm: 'Secret',
    inputType: 'autocomplete',
};

export const deploymentAttributes = [Annotation, ContainerType, ID, Label, Name, Inactive];
