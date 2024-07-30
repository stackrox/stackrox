// If you're adding a new attribute, make sure to add it to the "profileCheckAttributes" array as well

import { SearchFilterAttribute } from '../types';

const Name = {
    displayName: 'Name',
    filterChipLabel: 'Profile check name',
    searchTerm: 'Compliance Check Name',
    inputType: 'text',
} as const;

export const profileCheckAttributes = [Name] as const;

export type ProfileCheckAttribute = (typeof profileCheckAttributes)[number]['displayName'];

export function getProfileCheckAttributes(
    attributes?: ProfileCheckAttribute[]
): SearchFilterAttribute[] {
    if (!attributes || attributes.length === 0) {
        return profileCheckAttributes as unknown as SearchFilterAttribute[];
    }

    return profileCheckAttributes.filter((imageAttribute) => {
        return attributes.includes(imageAttribute.displayName);
    }) as SearchFilterAttribute[];
}
