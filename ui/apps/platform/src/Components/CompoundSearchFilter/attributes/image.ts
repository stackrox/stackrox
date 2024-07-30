import { SearchFilterAttribute } from '../types';

// If you're adding a new attribute, make sure to add it to the "imageAttributes" array as well

const Name = {
    displayName: 'Name',
    filterChipLabel: 'Image name',
    searchTerm: 'Image',
    inputType: 'autocomplete',
} as const;

const OperatingSystem = {
    displayName: 'Operating system',
    filterChipLabel: 'Image operating system',
    searchTerm: 'Image OS',
    inputType: 'autocomplete',
} as const;

const Tag = {
    displayName: 'Tag',
    filterChipLabel: 'Image tag',
    searchTerm: 'Image Tag',
    inputType: 'text',
} as const;

const Label = {
    displayName: 'Label',
    filterChipLabel: 'Image label',
    searchTerm: 'Image Label',
    inputType: 'autocomplete',
} as const;

const Registry = {
    displayName: 'Registry',
    filterChipLabel: 'Image registry',
    searchTerm: 'Image Registry',
    inputType: 'text',
} as const;

export const imageAttributes = [Name, OperatingSystem, Tag, Label, Registry] as const;

export type ImageAttribute = (typeof imageAttributes)[number]['displayName'];

export function getImageAttributes(attributes?: ImageAttribute[]): SearchFilterAttribute[] {
    if (!attributes || attributes.length === 0) {
        return imageAttributes as unknown as SearchFilterAttribute[];
    }

    return imageAttributes.filter((imageAttribute) => {
        return attributes.includes(imageAttribute.displayName);
    }) as SearchFilterAttribute[];
}
