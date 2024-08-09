// If you're adding a new attribute, make sure to add it to "policyAttributes" as well

import { CompoundSearchFilterAttribute } from '../types';

export const Name: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Policy name',
    searchTerm: 'Policy',
    inputType: 'autocomplete',
};

export const Category: CompoundSearchFilterAttribute = {
    displayName: 'Category',
    filterChipLabel: 'Policy category',
    searchTerm: 'Category',
    inputType: 'autocomplete',
};

export const LifecycleStage: CompoundSearchFilterAttribute = {
    displayName: 'Lifecycle stage',
    filterChipLabel: 'Lifecycle stage',
    searchTerm: 'Lifecycle Stage',
    inputType: 'select',
    inputProps: {
        options: [
            { value: 'DEPLOY', label: 'Deploy' },
            { value: 'BUILD', label: 'Build' },
            { value: 'RUNTIME', label: 'Runtime' },
        ],
    },
};

export const policyAttributes = [Name, Category, LifecycleStage];
