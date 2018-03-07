import axios from 'axios';

/**
 * Fetches list of registered notifiers.
 *
 * @returns {Promise<Object, Error>} fulfilled with array of notifiers (as defined in .proto)
 */
export function fetchNotifiers() {
    const notifiersUrl = '/v1/notifiers';
    return axios.get(notifiersUrl).then(response => ({
        response: response.data
    }));
}

/**
 * Fetches list of registered registries.
 *
 * @returns {Promise<Object, Error>} fulfilled with array of registries (as defined in .proto)
 */
export function fetchRegistries() {
    const registriesUrl = '/v1/registries';
    return axios.get(registriesUrl).then(response => ({
        response: response.data
    }));
}

/**
 * Fetches list of registered scanners.
 *
 * @returns {Promise<Object, Error>} fulfilled with array of scanners (as defined in .proto)
 */
export function fetchScanners() {
    const scannersUrl = '/v1/scanners';
    return axios.get(scannersUrl).then(response => ({
        response: response.data
    }));
}
