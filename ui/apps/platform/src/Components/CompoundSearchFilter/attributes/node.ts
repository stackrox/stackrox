// If you're adding a new attribute, make sure to add it to "nodeAttributes" as well

import { CompoundSearchFilterAttribute } from '../types';

export const Name: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Node name',
    searchTerm: 'Node',
    inputType: 'autocomplete',
};

export const OperatingSystem: CompoundSearchFilterAttribute = {
    displayName: 'Operating system',
    filterChipLabel: 'Node operating system',
    searchTerm: 'Operating System',
    inputType: 'text',
};

export const Label: CompoundSearchFilterAttribute = {
    displayName: 'Label',
    filterChipLabel: 'Node label',
    searchTerm: 'Node Label',
    inputType: 'autocomplete',
};

export const Annotation: CompoundSearchFilterAttribute = {
    displayName: 'Annotation',
    filterChipLabel: 'Node annotation',
    searchTerm: 'Node Annotation',
    inputType: 'autocomplete',
};

export const ScanTime: CompoundSearchFilterAttribute = {
    displayName: 'Scan time',
    filterChipLabel: 'Node scan time',
    searchTerm: 'Node Scan Time',
    inputType: 'date-picker',
};

export const nodeAttributes = [Name, OperatingSystem, Label, Annotation, ScanTime];
