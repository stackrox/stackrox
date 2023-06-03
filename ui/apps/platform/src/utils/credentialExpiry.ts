import { differenceInDays, distanceInWordsStrict, format } from 'date-fns';

import { CertExpiryComponent } from 'types/credentialExpiryService.proto';

export function getCredentialExpiryPhrase(expirationDate: string, currentDatetime: Date) {
    return `expires in ${distanceInWordsStrict(expirationDate, currentDatetime)} on ${format(
        expirationDate,
        'MMMM D, YYYY'
    )} at ${format(expirationDate, 'h:mm a')}`;
}

export const daysForCredentialExpiryDanger = 3; // red banner if less than or equal to 3 days
export const daysForCredentialExpiryWarning = 14; // yellow bannder if less than or equal to 14 days

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

export const nameOfComponent: Record<CertExpiryComponent, string> = {
    CENTRAL: 'Central',
    SCANNER: 'Scanner',
};
