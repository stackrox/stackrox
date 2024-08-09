// If you're adding a new attribute, make sure to add it to "alertAttributes" as well

import { CompoundSearchFilterAttribute } from '../types';

export const InactiveDeployment: CompoundSearchFilterAttribute = {
    displayName: 'Deployment status',
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

export const alertAttributes = [InactiveDeployment];
