import { Scope } from 'types/vuln_request.proto';
import { addDaysToDate } from 'utils/dateUtils';

export function getExpiresWhenFixedValue(expiresOn: string): boolean {
    return expiresOn === 'Until Fixable';
}

export type ExpiresOn = 'Until Fixable' | '2 weeks' | '30 days' | '90 days' | 'Indefinitely';

export function getExpiresOnValue(expiresOn: ExpiresOn): string | number | null {
    let value: string | number | null = null;
    if (expiresOn === '2 weeks') {
        value = addDaysToDate(new Date(), 14);
    } else if (expiresOn === '30 days') {
        value = addDaysToDate(new Date(), 30);
    } else if (expiresOn === '90 days') {
        value = addDaysToDate(new Date(), 90);
    } else if (expiresOn === 'Indefinitely') {
        value = null;
    }
    return value;
}

export function getScopeValue(
    imageAppliesTo: string,
    registry: string,
    remote: string,
    tag: string,
    digest: string,
): Scope {
    let value: Scope = { imageScope: undefined };
    if (imageAppliesTo === 'All tags within image') {
        value = {
            imageScope: {
                registry,
                remote,
                tag: '.*',
                digest: '.*',
            },
        };
    } else if (imageAppliesTo === 'Only this image tag') {
        value = {
            imageScope: {
                registry,
                remote,
                tag,
                digest: '.*',
            },
        };
    } else if (imageAppliesTo === 'Only this image digest') {
        value = {
            imageScope: {
                registry,
                remote,
                tag: '.*',
                digest,
            },
        };
    }
    return value;
}
