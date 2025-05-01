import { differenceInDays, distanceInWordsStrict } from 'date-fns';

import { CertExpiryComponent } from 'types/credentialExpiryService.proto';
import { getDateTime } from './dateUtils';

export function getCredentialExpiryPhrase(expirationDate: string, currentDatetime: Date) {
    return `expires in ${distanceInWordsStrict(expirationDate, currentDatetime)} on ${getDateTime(
        expirationDate
    )}`;
}

export const daysForCredentialExpiryDanger = 3; // red banner if less than or equal to 3 days
export const daysForCredentialExpiryWarning = 14; // yellow banner if less than or equal to 14 days

export type CredentialExpiryVariant = 'danger' | 'warning' | 'success';

export function getCredentialExpiryVariant(
    expirationDate: string,
    currentDatetime: Date
): CredentialExpiryVariant {
    const days = differenceInDays(expirationDate, currentDatetime);

    if (days <= daysForCredentialExpiryDanger) {
        return 'danger';
    }
    if (days <= daysForCredentialExpiryWarning) {
        return 'warning';
    }
    return 'success';
}

export function getBannerVariant(type: CredentialExpiryVariant) {
    switch (type) {
        case 'danger':
            return 'red';
        case 'warning':
            return 'gold';
        default:
            return 'green';
    }
}

export const nameOfComponent: Record<CertExpiryComponent, string> = {
    CENTRAL: 'Central',
    SCANNER: 'StackRox Scanner',
    SCANNER_V4: 'Scanner V4',
    CENTRAL_DB: 'Central Database',
};
