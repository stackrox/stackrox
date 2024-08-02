// If you're adding a new attribute, make sure to add it to the "namespaceAttributes" object as well

export const ID = {
    displayName: 'ID',
    filterChipLabel: 'Namespace ID',
    searchTerm: 'Namespace ID',
    inputType: 'autocomplete',
} as const;

export const Name = {
    displayName: 'Name',
    filterChipLabel: 'Namespace name',
    searchTerm: 'Namespace',
    inputType: 'autocomplete',
} as const;

export const Label = {
    displayName: 'Label',
    filterChipLabel: 'Namespace label',
    searchTerm: 'Namespace Label',
    inputType: 'autocomplete',
} as const;

export const Annotation = {
    displayName: 'Annotation',
    filterChipLabel: 'Namespace annotation',
    searchTerm: 'Namespace Annotation',
    inputType: 'autocomplete',
} as const;

export const namespaceAttributes = { ID, Name, Label, Annotation } as const;
