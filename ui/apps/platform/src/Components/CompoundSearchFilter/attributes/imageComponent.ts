// If you're adding a new attribute, make sure to add it to "imageComponentAttributes" as well

import { sourceTypeLabels, sourceTypes } from 'types/image.proto';
import { CompoundSearchFilterAttribute } from '../types';

export const Name: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Image component name',
    searchTerm: 'Component',
    inputType: 'autocomplete',
};

export const Source: CompoundSearchFilterAttribute = {
    displayName: 'Source',
    filterChipLabel: 'Image component source',
    searchTerm: 'Component Source',
    inputType: 'select',
    inputProps: {
        options: sourceTypes.map((sourceType) => {
            return { label: sourceTypeLabels[sourceType], value: sourceType };
        }),
    },
};

export const Version: CompoundSearchFilterAttribute = {
    displayName: 'Version',
    filterChipLabel: 'Image component version',
    searchTerm: 'Component Version',
    inputType: 'text',
};

export const imageComponentAttributes = [Name, Source, Version];
