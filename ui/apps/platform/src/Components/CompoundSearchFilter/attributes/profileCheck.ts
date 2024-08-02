// If you're adding a new attribute, make sure to add it to "profileCheckAttributes" as well

import { CompoundSearchFilterAttribute } from '../types';

export const Name: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Profile check name',
    searchTerm: 'Compliance Check Name',
    inputType: 'text',
};

export const profileCheckAttributes = [Name];
