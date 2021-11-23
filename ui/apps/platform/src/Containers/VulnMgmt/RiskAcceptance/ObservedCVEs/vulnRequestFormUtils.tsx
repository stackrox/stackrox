import { Scope } from 'types/vuln_request.proto';
import { addDaysToDate } from 'utils/dateUtils';

export function getExpiresWhenFixedValue(expiresOn: string): boolean {
    return expiresOn === 'Until Fixable';
}

export function getExpiresOnValue(expiresOn: string): string | undefined {
    let value: string | undefined;
    if (expiresOn === '2 weeks') {
        value = addDaysToDate(new Date(), 14);
    } else if (expiresOn === '30 days') {
        value = addDaysToDate(new Date(), 30);
    } else if (expiresOn === '90 days') {
        value = addDaysToDate(new Date(), 90);
    } else if (expiresOn === 'Indefinitely') {
        // @TODO: This should be changed to 0 once Mandar's changes are in
        value = addDaysToDate(new Date(), 18250);
    }
    return value;
}

export function getScopeValue(imageAppliesTo: string, imageName: string, tag: string): Scope {
    let value: Scope = { imageScope: undefined };
    if (imageAppliesTo === 'All tags within image') {
        value = {
            imageScope: {
                name: imageName,
                tagRegex: '.*',
            },
        };
    } else if (imageAppliesTo === 'Only this image tag') {
        value = {
            imageScope: {
                name: imageName,
                tagRegex: tag,
            },
        };
    }
    return value;
}
