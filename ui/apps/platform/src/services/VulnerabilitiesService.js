import entityTypes from 'constants/entityTypes';
import axios from './instance';

function getBaseCveUrl(cveType) {
    if (cveType === entityTypes.CLUSTER_CVE) {
        return '/v1/clustercves';
    }
    if (cveType === entityTypes.NODE_CVE) {
        return '/v1/nodecves';
    }
    if (cveType === entityTypes.IMAGE_CVE) {
        return '/v1/imagecves';
    }
    return '/v1/cves';
}

/**
 * Send request to suppress CVEs with a given IDs.
 *
 * @param {string} cveType The type of CVEs to suppress
 * @param {string[]} cveNames CVE names to suppress
 * @param {string} duration CVE suppress duration, in hours, if "0" then CVEs are suppressed indefinitely
 * @returns {Promise<AxiosResponse, Error>} fulfilled in case of success or rejected with an error
 */
export function suppressVulns(cveType, cveNames, duration = '0') {
    const baseUrl = getBaseCveUrl(cveType);
    return axios.patch(`${baseUrl}/suppress`, { cves: cveNames, duration });
}

/**
 * Send request to unsuppress CVEs with a given IDs.
 *
 * @param {string} cveType The type of CVEs to suppress
 * @param {string[]} cveNames CVE names to suppress
 * @returns {Promise<AxiosResponse, Error>} fulfilled in case of success or rejected with an error
 */
export function unsuppressVulns(cveType, cveNames) {
    const baseUrl = getBaseCveUrl(cveType);
    return axios.patch(`${baseUrl}/unsuppress`, { cves: cveNames });
}
