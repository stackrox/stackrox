import axios from './instance';

const credentialExpiryBaseURL = '/v1/credentialexpiry';

export function fetchCentralCertExpiry() {
    return axios
        .get(`${credentialExpiryBaseURL}?component=CENTRAL`)
        .then((response) => response?.data?.expiry ?? '');
}

export function fetchScannerCertExpiry() {
    return axios
        .get(`${credentialExpiryBaseURL}?component=SCANNER`)
        .then((response) => response?.data?.expiry ?? '');
}
