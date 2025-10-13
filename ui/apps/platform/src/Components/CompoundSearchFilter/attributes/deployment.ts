// If you're adding a new attribute, make sure to add it to "deploymentAttributes" as well

import { CompoundSearchFilterAttribute } from '../types';

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

export const deploymentAttributes = [ID, Name, Label, Annotation, Inactive];
