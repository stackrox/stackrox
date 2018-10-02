import { normalize } from 'normalizr';
import axios from 'axios';
import queryString from 'query-string';

import { alert as alertSchema, alerts as alertsSchema } from './schemas';

const baseUrl = '/v1/alerts';

/**
 * Fetches alert details for a given alert ID.
 * Returns normalized response with alert entity extracted.
 *
 * @param {!string} alertId
 * @returns {Promise<Object, Error>} fulfilled with normalized response
 */
export function fetchAlerts(filters) {
    const params = queryString.stringify({
        ...filters
    });
    return axios.get(`${baseUrl}?${params}`).then(response => ({
        response: normalize(response.data, alertsSchema)
    }));
}

/**
 * Fetches alert details for a given alert ID.
 * Returns normalized response with alert entity extracted.
 *
 * @param {!string} alertId
 * @returns {Promise<Object, Error>} fulfilled with normalized response
 */
export function fetchAlert(alertId) {
    return axios.get(`${baseUrl}/${alertId}`).then(response => ({
        response: normalize(response.data, alertSchema)
    }));
}

/**
 * Fetches severity counts
 *
 * @returns {Promise<Object, Error>}
 */
export function fetchAlertCounts(filters) {
    const params = queryString.stringify({
        ...filters
    });
    return axios.get(`${baseUrl}/summary/counts?${params}`).then(response => ({
        response: response.data
    }));
}

/**
 * Fetches alerts by time for timeseries.
 *
 * @returns {Promise<Object, Error>}
 */
export function fetchAlertsByTimeseries(filters) {
    const params = queryString.stringify({
        ...filters
    });
    return axios.get(`${baseUrl}/summary/timeseries?${params}`).then(response => ({
        response: response.data
    }));
}

/**
 * Resolves an alert given an alert ID.
 *
 * @param {!string} alertId
 * @returns {Promise<AxiosResponse, Error>}
 */
export function resolveAlert(alertId) {
    return axios.patch(`${baseUrl}/${alertId}/resolve`);
}

/**
 * Resolves a list of alerts by alert ID.
 *
 * @param {string[]} alertIds
 * @returns {Promise<AxiosResponse, Error>}
 */
export function resolveAlerts(alertIds = []) {
    return Promise.all(alertIds.map(id => resolveAlert(id)));
}
