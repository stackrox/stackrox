// If you're adding a new attribute, make sure to add it to the "nodeAttributes" object as well

export const Name = {
    displayName: 'Name',
    filterChipLabel: 'Node name',
    searchTerm: 'Node',
    inputType: 'autocomplete',
} as const;

export const OperatingSystem = {
    displayName: 'Operating system',
    filterChipLabel: 'Node operating system',
    searchTerm: 'Operating System',
    inputType: 'text',
} as const;

export const Label = {
    displayName: 'Label',
    filterChipLabel: 'Node label',
    searchTerm: 'Node Label',
    inputType: 'autocomplete',
} as const;

export const Annotation = {
    displayName: 'Annotation',
    filterChipLabel: 'Node annotation',
    searchTerm: 'Node Annotation',
    inputType: 'autocomplete',
} as const;

export const ScanTime = {
    displayName: 'Scan time',
    filterChipLabel: 'Node scan time',
    searchTerm: 'Node Scan Time',
    inputType: 'date-picker',
} as const;

export const nodeAttributes = { Name, OperatingSystem, Label, Annotation, ScanTime } as const;
