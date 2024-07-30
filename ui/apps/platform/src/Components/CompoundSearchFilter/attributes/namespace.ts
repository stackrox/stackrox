// If you're adding a new attribute, make sure to add it to the "namespaceAttributes" array as well

import { SearchFilterAttribute } from '../types';

const ID = {
    displayName: 'ID',
    filterChipLabel: 'Namespace ID',
    searchTerm: 'Namespace ID',
    inputType: 'autocomplete',
} as const;

const Name = {
    displayName: 'Name',
    filterChipLabel: 'Namespace name',
    searchTerm: 'Namespace',
    inputType: 'autocomplete',
} as const;

const Label = {
    displayName: 'Label',
    filterChipLabel: 'Namespace label',
    searchTerm: 'Namespace Label',
    inputType: 'autocomplete',
} as const;

const Annotation = {
    displayName: 'Annotation',
    filterChipLabel: 'Namespace annotation',
    searchTerm: 'Namespace Annotation',
    inputType: 'autocomplete',
} as const;

export const namespaceAttributes = [ID, Name, Label, Annotation] as const;

export type NamespaceAttribute = (typeof namespaceAttributes)[number]['displayName'];

export function getNamespaceAttributes(attributes?: NamespaceAttribute[]): SearchFilterAttribute[] {
    if (!attributes || attributes.length === 0) {
        return namespaceAttributes as unknown as SearchFilterAttribute[];
    }

    return namespaceAttributes.filter((imageAttribute) => {
        return attributes.includes(imageAttribute.displayName);
    }) as SearchFilterAttribute[];
}
