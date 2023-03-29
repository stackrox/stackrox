import { Traits } from '../../types/traits.proto';

export function isUserResource(traits?: Traits): boolean {
    return traits == null || traits.origin == null || traits.origin === 'IMPERATIVE';
}

export const traitsOriginLabels = {
    DEFAULT: 'System',
    IMPERATIVE: 'User',
    DECLARATIVE: 'Declarative',
};

export const originLabelColours = {
    System: 'grey',
    User: 'green',
    Declarative: 'blue',
};

export function getOriginLabel(traits?: Traits): string {
    return traits && traits.origin && traitsOriginLabels[traits.origin]
        ? traitsOriginLabels[traits.origin]
        : 'User';
}
