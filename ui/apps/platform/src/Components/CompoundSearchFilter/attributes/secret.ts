// If you're adding a new attribute, make sure to add it to "secretAttributes" as well

import type { CompoundSearchFilterAttribute } from '../types';

export const Name: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Secret name',
    searchTerm: 'Secret',
    inputType: 'autocomplete',
};

export const secretAttributes = [Name];
