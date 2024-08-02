// If you're adding a new attribute, make sure to add it to "nodeComponentAttributes" as well

import { CompoundSearchFilterAttribute } from '../types';

export const Name: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Image component name',
    searchTerm: 'Component',
    inputType: 'autocomplete',
};

export const Version: CompoundSearchFilterAttribute = {
    displayName: 'Version',
    filterChipLabel: 'Image component version',
    searchTerm: 'Component Version',
    inputType: 'text',
};

export const nodeComponentAttributes = [Name, Version];
