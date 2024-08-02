import { sourceTypeLabels, sourceTypes } from 'types/image.proto';

// If you're adding a new attribute, make sure to add it to the "imageComponentAttributes" object as well

export const Name = {
    displayName: 'Name',
    filterChipLabel: 'Image component name',
    searchTerm: 'Component',
    inputType: 'autocomplete',
} as const;

export const Source = {
    displayName: 'Source',
    filterChipLabel: 'Image component source',
    searchTerm: 'Component Source',
    inputType: 'select',
    inputProps: {
        options: sourceTypes.map((sourceType) => {
            return { label: sourceTypeLabels[sourceType], value: sourceType };
        }),
    },
} as const;

export const Version = {
    displayName: 'Version',
    filterChipLabel: 'Image component version',
    searchTerm: 'Component Version',
    inputType: 'text',
} as const;

export const imageComponentAttributes = { Name, Source, Version } as const;
