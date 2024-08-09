// If you're adding a new attribute, make sure to add it to "alertAttributes" as well

import { CompoundSearchFilterAttribute } from '../types';

export const InactiveDeployment: CompoundSearchFilterAttribute = {
    displayName: 'Inactive deployment',
    filterChipLabel: 'Inactive deployment',
    searchTerm: 'Inactive Deployment',
    inputType: 'select',
    inputProps: {
        options: [
            { value: 'true', label: 'True' },
            { value: 'false', label: 'False' },
        ],
    },
};

export const alertAttributes = [InactiveDeployment];
