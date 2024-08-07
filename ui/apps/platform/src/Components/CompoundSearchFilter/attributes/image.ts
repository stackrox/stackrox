// If you're adding a new attribute, make sure to add it to "imageAttributes" as well

import { CompoundSearchFilterAttribute } from '../types';

export const Name: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Image name',
    searchTerm: 'Image',
    inputType: 'autocomplete',
};

export const OperatingSystem: CompoundSearchFilterAttribute = {
    displayName: 'Operating system',
    filterChipLabel: 'Image operating system',
    searchTerm: 'Image OS',
    inputType: 'autocomplete',
};

export const Tag: CompoundSearchFilterAttribute = {
    displayName: 'Tag',
    filterChipLabel: 'Image tag',
    searchTerm: 'Image Tag',
    inputType: 'text',
};

export const Label: CompoundSearchFilterAttribute = {
    displayName: 'Label',
    filterChipLabel: 'Image label',
    searchTerm: 'Image Label',
    inputType: 'autocomplete',
};

export const Registry: CompoundSearchFilterAttribute = {
    displayName: 'Registry',
    filterChipLabel: 'Image registry',
    searchTerm: 'Image Registry',
    inputType: 'text',
};

export const imageAttributes = [Name, OperatingSystem, Tag, Label, Registry];
