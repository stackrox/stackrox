import type { Traits } from 'types/traits.proto';

export function isUserResource(traits?: Traits | null): boolean {
    return traits == null || traits.origin == null || traits.origin === 'IMPERATIVE';
}

export const traitsOriginLabels = {
    DEFAULT: 'System',
    IMPERATIVE: 'User',
    DECLARATIVE: 'Declarative',
    DECLARATIVE_ORPHANED: 'Declarative, Orphaned',
} as const;

export const originLabelColours = {
    System: 'grey',
    User: 'green',
    Declarative: 'blue',
    'Declarative, Orphaned': 'red',
} as const;

export type OriginLabel = (typeof traitsOriginLabels)[keyof typeof traitsOriginLabels];

export function getOriginLabel(traits?: Traits | null): OriginLabel {
    return traits && traits.origin && traitsOriginLabels[traits.origin]
        ? traitsOriginLabels[traits.origin]
        : 'User';
}
