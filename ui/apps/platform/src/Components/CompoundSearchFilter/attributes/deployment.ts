// If you're adding a new attribute, make sure to add it to the "deploymentAttributes" object as well

export const ID = {
    displayName: 'ID',
    filterChipLabel: 'Deployment ID',
    searchTerm: 'Deployment ID',
    inputType: 'autocomplete',
} as const;

export const Name = {
    displayName: 'Name',
    filterChipLabel: 'Deployment name',
    searchTerm: 'Deployment',
    inputType: 'autocomplete',
} as const;

export const Label = {
    displayName: 'Label',
    filterChipLabel: 'Deployment label',
    searchTerm: 'Deployment Label',
    inputType: 'autocomplete',
} as const;

export const Annotation = {
    displayName: 'Annotation',
    filterChipLabel: 'Deployment annotation',
    searchTerm: 'Deployment Annotation',
    inputType: 'autocomplete',
} as const;

export const deploymentAttributes = { ID, Name, Label, Annotation } as const;
