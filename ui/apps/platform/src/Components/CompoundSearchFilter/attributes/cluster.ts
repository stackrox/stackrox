// If you're adding a new attribute, make sure to add it to the "clusterAttributes" object as well

export const ID = {
    displayName: 'ID',
    filterChipLabel: 'Cluster ID',
    searchTerm: 'Cluster ID',
    inputType: 'autocomplete',
} as const;

export const Name = {
    displayName: 'Name',
    filterChipLabel: 'Cluster name',
    searchTerm: 'Cluster',
    inputType: 'autocomplete',
} as const;

export const Label = {
    displayName: 'Label',
    filterChipLabel: 'Cluster label',
    searchTerm: 'Cluster Label',
    inputType: 'autocomplete',
} as const;

export const Type = {
    displayName: 'Type',
    filterChipLabel: 'Cluster type',
    searchTerm: 'Cluster Type',
    inputType: 'autocomplete',
} as const;

export const PlatformType = {
    displayName: 'Platform Type',
    filterChipLabel: 'Platform type',
    searchTerm: 'Cluster Platform Type',
    inputType: 'autocomplete',
} as const;

export const clusterAttributes = { ID, Name, Label, Type, PlatformType } as const;
