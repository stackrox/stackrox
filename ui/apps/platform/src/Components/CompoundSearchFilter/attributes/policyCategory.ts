// If you're adding a new attribute, make sure to add it to "policyCategoryAttributes" as well

import type { CompoundSearchFilterAttribute } from '../types';

export const Name: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Policy category name',
    searchTerm: 'Policy Category',
    inputType: 'autocomplete',
};

export const policyCategoryAttributes = [Name];
