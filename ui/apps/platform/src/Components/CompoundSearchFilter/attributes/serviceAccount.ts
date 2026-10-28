// If you're adding a new attribute, make sure to add it to "serviceAccountAttributes" as well

import type { CompoundSearchFilterAttribute } from '../types';

export const Name: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Service account name',
    searchTerm: 'Service Account',
    inputType: 'autocomplete',
};

export const serviceAccountAttributes = [Name];
