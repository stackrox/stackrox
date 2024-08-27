// If you're adding a new attribute, make sure to add it to "resourceAttributes" as well

import { CompoundSearchFilterAttribute } from '../types';

export const Name: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Resource name',
    searchTerm: 'Resource',
    inputType: 'autocomplete',
};

export const alertAttributes = [Name];
