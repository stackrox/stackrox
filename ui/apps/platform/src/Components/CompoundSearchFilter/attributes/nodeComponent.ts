// If you're adding a new attribute, make sure to add it to the "nodeComponentAttributes" array as well

import { SearchFilterAttribute } from '../types';

const Name = {
    displayName: 'Name',
    filterChipLabel: 'Image component name',
    searchTerm: 'Component',
    inputType: 'autocomplete',
} as const;

const Version = {
    displayName: 'Version',
    filterChipLabel: 'Image component version',
    searchTerm: 'Component Version',
    inputType: 'text',
} as const;

export const nodeComponentAttributes = [Name, Version] as const;

export type NodeComponentAttribute = (typeof nodeComponentAttributes)[number]['displayName'];

export function getNodeComponentAttributes(
    attributes?: NodeComponentAttribute[]
): SearchFilterAttribute[] {
    if (!attributes || attributes.length === 0) {
        return nodeComponentAttributes as unknown as SearchFilterAttribute[];
    }

    return nodeComponentAttributes.filter((imageAttribute) => {
        return attributes.includes(imageAttribute.displayName);
    }) as SearchFilterAttribute[];
}
