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

export const EnforcementAction: CompoundSearchFilterAttribute = {
    displayName: 'Enforcement action',
    filterChipLabel: 'Enforcement action',
    searchTerm: 'Enforcement',
    inputType: 'select',
    inputProps: {
        options: [
            { value: 'UNSET_ENFORCEMENT', label: 'Unset Enforcement' },
            { value: 'SCALE_TO_ZERO_ENFORCEMENT', label: 'Scale to zero enforcement' },
            {
                value: 'UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT',
                label: 'Unsatisfiable node constraint enforcement',
            },
            { value: 'KILL_POD_ENFORCEMENT', label: 'Kill pod enforcement' },
            { value: 'FAIL_BUILD_ENFORCEMENT', label: 'Fail build enforcement' },
            { value: 'FAIL_KUBE_REQUEST_ENFORCEMENT', label: 'Fail kube request enforcement' },
            {
                value: 'FAIL_DEPLOYMENT_CREATE_ENFORCEMENT',
                label: 'Fail deployment create enforcement',
            },
            {
                value: 'FAIL_DEPLOYMENT_UPDATE_ENFORCEMENT',
                label: 'Fail deployment update enforcement',
            },
        ],
    },
};

export const policyAttributes = [Name, Category, EnforcementAction];
