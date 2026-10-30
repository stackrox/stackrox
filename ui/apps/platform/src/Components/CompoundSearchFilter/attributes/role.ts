// If you're adding a new attribute, make sure to add it to "roleAttributes" as well

import type { CompoundSearchFilterAttribute } from '../types';

export const Name: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Role name',
    searchTerm: 'Role',
    inputType: 'autocomplete',
};

export const roleAttributes = [Name];
