// If you're adding a new attribute, make sure to add it to "namespaceAttributes" as well

import { CompoundSearchFilterAttribute } from '../types';

export const ID: CompoundSearchFilterAttribute = {
    displayName: 'ID',
    filterChipLabel: 'Namespace ID',
    searchTerm: 'Namespace ID',
    inputType: 'autocomplete',
};

export const Name: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Namespace name',
    searchTerm: 'Namespace',
    inputType: 'autocomplete',
};

export const Label: CompoundSearchFilterAttribute = {
    displayName: 'Label',
    filterChipLabel: 'Namespace label',
    searchTerm: 'Namespace Label',
    inputType: 'autocomplete',
};

export const Annotation: CompoundSearchFilterAttribute = {
    displayName: 'Annotation',
    filterChipLabel: 'Namespace annotation',
    searchTerm: 'Namespace Annotation',
    inputType: 'autocomplete',
};

export const namespaceAttributes = [ID, Name, Label, Annotation];
