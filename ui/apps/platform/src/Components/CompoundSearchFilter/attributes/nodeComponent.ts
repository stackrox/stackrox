// If you're adding a new attribute, make sure to add it to the "nodeComponentAttributes" object as well

export const Name = {
    displayName: 'Name',
    filterChipLabel: 'Image component name',
    searchTerm: 'Component',
    inputType: 'autocomplete',
} as const;

export const Version = {
    displayName: 'Version',
    filterChipLabel: 'Image component version',
    searchTerm: 'Component Version',
    inputType: 'text',
} as const;

export const nodeComponentAttributes = { Name, Version } as const;
