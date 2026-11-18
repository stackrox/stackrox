// If you're adding a new attribute, make sure to add it to "roleBindingAttributes" as well

import type { CompoundSearchFilterAttribute } from '../types';

export const Name: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Role binding name',
    searchTerm: 'Role Binding',
    inputType: 'autocomplete',
};

export const roleBindingAttributes = [Name];
