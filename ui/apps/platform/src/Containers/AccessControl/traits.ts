import { Traits } from '../../types/traits.proto';

export function isUserResource(traits?: Traits | null): boolean {
    return traits == null || traits.origin == null || traits.origin === 'IMPERATIVE';
}

export const traitsOriginLabels = {
    DEFAULT: 'System',
    IMPERATIVE: 'User',
    DECLARATIVE: 'Declarative',
    DECLARATIVE_ORPHANED: 'Declarative, Orphaned',
};

export const originLabelColours = {
    System: 'grey',
    User: 'green',
    Declarative: 'blue',
    'Declarative, Orphaned': 'red',
};

export function getOriginLabel(traits?: Traits | null): string {
    return traits && traits.origin && traitsOriginLabels[traits.origin]
        ? traitsOriginLabels[traits.origin]
        : 'User';
}
