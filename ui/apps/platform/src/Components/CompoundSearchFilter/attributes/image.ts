// If you're adding a new attribute, make sure to add it to the "imageAttributes" object as well

export const Name = {
    displayName: 'Name',
    filterChipLabel: 'Image name',
    searchTerm: 'Image',
    inputType: 'autocomplete',
} as const;

export const OperatingSystem = {
    displayName: 'Operating system',
    filterChipLabel: 'Image operating system',
    searchTerm: 'Image OS',
    inputType: 'autocomplete',
} as const;

export const Tag = {
    displayName: 'Tag',
    filterChipLabel: 'Image tag',
    searchTerm: 'Image Tag',
    inputType: 'text',
} as const;

export const Label = {
    displayName: 'Label',
    filterChipLabel: 'Image label',
    searchTerm: 'Image Label',
    inputType: 'autocomplete',
} as const;

export const Registry = {
    displayName: 'Registry',
    filterChipLabel: 'Image registry',
    searchTerm: 'Image Registry',
    inputType: 'text',
} as const;

export const imageAttributes = { Name, OperatingSystem, Tag, Label, Registry } as const;
