// If you're adding a new attribute, make sure to add it to "imageComponentAttributes" as well

import { sourceTypeLabels, sourceTypes } from 'types/image.proto';
import type { CompoundSearchFilterAttribute } from '../types';

export const Name: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Image component name',
    searchTerm: 'Component',
    inputType: 'autocomplete',
};

export const Source: CompoundSearchFilterAttribute = {
    displayName: 'Source',
    filterChipLabel: 'Image component source',
    searchTerm: 'Component Source',
    inputType: 'select',
    inputProps: {
        options: sourceTypes.map((sourceType) => {
            return { label: sourceTypeLabels[sourceType], value: sourceType };
        }),
    },
};

export const Version: CompoundSearchFilterAttribute = {
    displayName: 'Version',
    filterChipLabel: 'Image component version',
    searchTerm: 'Component Version',
    inputType: 'text',
};

export const LayerType: CompoundSearchFilterAttribute = {
    displayName: 'Layer type',
    filterChipLabel: 'Image component layer type',
    searchTerm: 'Component From Base Image',
    inputType: 'select',
    featureFlagDependency: ['ROX_BASE_IMAGE_DETECTION'],
    inputProps: {
        options: [
            { label: 'Application', value: 'false' },
            { label: 'Base image', value: 'true' },
        ],
    },
};

export const imageComponentAttributes = [LayerType, Name, Source, Version];
