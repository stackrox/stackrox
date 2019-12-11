import axios from './instance';

const baseUrl = '/v1/cves';

/**
 * Send request to suppress / unsuppress CVE with a given ID.
 *
 * @param {!string} CVE unique identifier
 * @param {!boolean} true if CVE should be suppressed, false for unsuppress
 * @returns {Promise<AxiosResponse, Error>} fulfilled in case of success or rejected with an error
 */
// eslint-disable-next-line import/prefer-default-export
export function updateCveSuppressedState(cve, suppressed = false) {
    return axios.patch(`${baseUrl}/${cve}`, { suppressed });
}
