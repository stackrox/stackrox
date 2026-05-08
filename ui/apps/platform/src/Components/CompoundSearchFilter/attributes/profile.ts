import type { CompoundSearchFilterAttribute } from '../types';

export const profileName: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Profile name',
    searchTerm: 'Compliance Profile Name',
    inputType: 'text',
};

export const profileType: CompoundSearchFilterAttribute = {
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
