import { sourceTypeLabels, sourceTypes } from 'types/image.proto';
import { SearchFilterAttribute } from '../types';

// If you're adding a new attribute, make sure to add it to the "imageComponentAttributes" array as well

const Name = {
    displayName: 'Name',
    filterChipLabel: 'Image component name',
    searchTerm: 'Component',
    inputType: 'autocomplete',
} as const;

const Source = {
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

const Version = {
    displayName: 'Version',
    filterChipLabel: 'Image component version',
    searchTerm: 'Component Version',
    inputType: 'text',
} as const;

export const imageComponentAttributes = [Name, Source, Version] as const;

export type ImageComponentAttribute = (typeof imageComponentAttributes)[number]['displayName'];

export function getImageComponentAttributes(
    attributes?: ImageComponentAttribute[]
): SearchFilterAttribute[] {
    if (!attributes || attributes.length === 0) {
        return imageComponentAttributes as unknown as SearchFilterAttribute[];
    }

    return imageComponentAttributes.filter((imageAttribute) => {
        return attributes.includes(imageAttribute.displayName);
    }) as SearchFilterAttribute[];
}
