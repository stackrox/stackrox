// If you're adding a new attribute, make sure to add it to the "deploymentAttributes" array as well

import { SearchFilterAttribute } from '../types';

const ID = {
    displayName: 'ID',
    filterChipLabel: 'Deployment ID',
    searchTerm: 'Deployment ID',
    inputType: 'autocomplete',
} as const;

const Name = {
    displayName: 'Name',
    filterChipLabel: 'Deployment name',
    searchTerm: 'Deployment',
    inputType: 'autocomplete',
} as const;

const Label = {
    displayName: 'Label',
    filterChipLabel: 'Deployment label',
    searchTerm: 'Deployment Label',
    inputType: 'autocomplete',
} as const;

const Annotation = {
    displayName: 'Annotation',
    filterChipLabel: 'Deployment annotation',
    searchTerm: 'Deployment Annotation',
    inputType: 'autocomplete',
} as const;

export const deploymentAttributes = [ID, Name, Label, Annotation] as const;

export type DeploymentAttribute = (typeof deploymentAttributes)[number]['displayName'];

export function getDeploymentAttributes(
    attributes?: DeploymentAttribute[]
): SearchFilterAttribute[] {
    if (!attributes || attributes.length === 0) {
        return deploymentAttributes as unknown as SearchFilterAttribute[];
    }

    return deploymentAttributes.filter((imageAttribute) => {
        return attributes.includes(imageAttribute.displayName);
    }) as SearchFilterAttribute[];
}
