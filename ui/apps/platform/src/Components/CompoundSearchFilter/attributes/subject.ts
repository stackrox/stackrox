// If you're adding a new attribute, make sure to add it to "subjectAttributes" as well

import type { CompoundSearchFilterAttribute } from '../types';

export const Name: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'User or group name',
    searchTerm: 'Subject',
    inputType: 'autocomplete',
};

export const subjectAttributes = [Name];
