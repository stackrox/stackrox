// If you're adding a new attribute, make sure to add it to the "profileCheckAttributes" object as well

export const Name = {
    displayName: 'Name',
    filterChipLabel: 'Profile check name',
    searchTerm: 'Compliance Check Name',
    inputType: 'text',
} as const;

export const profileCheckAttributes = { Name } as const;
