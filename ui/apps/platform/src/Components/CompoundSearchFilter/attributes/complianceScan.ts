// If you're adding a new attribute, make sure to add it to the "complianceScanAttributes" object as well

export const ConfigID = {
    displayName: 'Config ID',
    filterChipLabel: 'Compliance scan config ID',
    searchTerm: 'Compliance Scan Config Id',
    inputType: 'text',
} as const;

export const complianceScanAttributes = { ConfigID } as const;
