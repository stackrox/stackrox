// If you're adding a new attribute, make sure to add it to "alertAttributes" as well

import { CompoundSearchFilterAttribute } from '../types';

export const ViolationTime: CompoundSearchFilterAttribute = {
    displayName: 'Violation time',
    filterChipLabel: 'Violation time',
    searchTerm: 'Violation Time',
    inputType: 'date-picker',
};

export const alertAttributes = [ViolationTime];
