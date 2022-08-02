import { CertExpiryComponent } from 'types/credentialExpiryService.proto';

import axios from './instance';

const credentialExpiryBaseURL = '/v1/credentialexpiry';

/**
 * Return ISO 8601 date string.
 */
export function fetchCertExpiryForComponent(component: CertExpiryComponent): Promise<string> {
    return axios
        .get<{ expiry: string }>(`${credentialExpiryBaseURL}?component=${component}`)
        .then((response) => response?.data?.expiry ?? '');
}
