// If you're adding a new attribute, make sure to add it to the "nodeAttributes" array as well

import { SearchFilterAttribute } from '../types';

const Name = {
    displayName: 'Name',
    filterChipLabel: 'Node name',
    searchTerm: 'Node',
    inputType: 'autocomplete',
} as const;

const OperatingSystem = {
    displayName: 'Operating system',
    filterChipLabel: 'Node operating system',
    searchTerm: 'Operating System',
    inputType: 'text',
} as const;

const Label = {
    displayName: 'Label',
    filterChipLabel: 'Node label',
    searchTerm: 'Node Label',
    inputType: 'autocomplete',
} as const;

const Annotation = {
    displayName: 'Annotation',
    filterChipLabel: 'Node annotation',
    searchTerm: 'Node Annotation',
    inputType: 'autocomplete',
} as const;

const ScanTime = {
    displayName: 'Scan time',
    filterChipLabel: 'Node scan time',
    searchTerm: 'Node Scan Time',
    inputType: 'date-picker',
} as const;

export const nodeAttributes = [Name, OperatingSystem, Label, Annotation, ScanTime] as const;

export type NodeAttribute = (typeof nodeAttributes)[number]['displayName'];

export function getNodeAttributes(attributes?: NodeAttribute[]): SearchFilterAttribute[] {
    if (!attributes || attributes.length === 0) {
        return nodeAttributes as unknown as SearchFilterAttribute[];
    }

    return nodeAttributes.filter((imageAttribute) => {
        return attributes.includes(imageAttribute.displayName);
    }) as SearchFilterAttribute[];
}
