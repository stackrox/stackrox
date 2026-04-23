import type { CompoundSearchFilterAttribute } from '../types';

export const Name: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Profile name',
    searchTerm: 'Compliance Profile Name',
    inputType: 'text',
};

export const Type: CompoundSearchFilterAttribute = {
    displayName: 'Type',
    filterChipLabel: 'Profile type',
    searchTerm: 'Compliance Profile Operator Kind',
    inputType: 'select',
    inputProps: {
        options: [
            { label: 'Built-in', value: 'PROFILE' },
            { label: 'Tailored', value: 'TAILORED_PROFILE' },
        ],
    },
};

export const profileAttributes = [Name, Type];
