import queryString from 'qs';
import searchOptionsToQuery from './searchOptionsToQuery';

import axios from './instance';

const baseUrl = '/v1/alerts';
const baseCountUrl = '/v1/alertscount';

/**
 * Fetches alerts by time for timeseries.
 *
 * @returns {Promise<Object, Error>}
 */
export function fetchAlertsByTimeseries(filters) {
    const params = queryString.stringify({
        ...filters,
    });

    // set higher timeout for this call to handle known backend scale issues with dashboard
    return axios
        .get(`${baseUrl}/summary/timeseries?${params}`, { timeout: 59999 })
        .then((response) => ({
            response: response.data,
        }));
}

/**
 * Fetches severity counts
 *
 * @returns {Promise<Object, Error>}
 */
export function fetchSummaryAlertCounts(filters) {
    const params = queryString.stringify({
        ...filters,
    });

    // set higher timeout for this call to handle known backend scale issues with dashboard
    return axios
        .get(`${baseUrl}/summary/counts?${params}`, { timeout: 59999 })
        .then((response) => ({
            response: response.data,
        }));
}

/**
 * Fetches a page of list alert objects.
 *
 * @param {!array} options
 * @param {!number} page
 * @param {!number} pageSize
 * @returns {Promise<Object, Error>} fulfilled with normalized response
 */
export function fetchAlerts(options, sortOption, page, pageSize) {
    const offset = page > 0 ? page * pageSize : 0;
    const query = searchOptionsToQuery(options);
    const params = queryString.stringify(
        {
            query,
            pagination: {
                offset,
                limit: pageSize,
                sortOption,
            },
        },
        { arrayFormat: 'repeat', allowDots: true }
    );
    return axios
        .get(`${baseUrl}?${params}`)
        .then((response) => (response.data ? response.data.alerts : []));
}

/**
 * Fetches list of count of alerts, using the input hooks to give the results.
 *
 * @param {!array} options
 * @returns {Promise<Object, Error>} fulfilled with normalized response
 */
export function fetchAlertCount(options) {
    const params = queryString.stringify(
        { query: searchOptionsToQuery(options) },
        { arrayFormat: 'repeat' }
    );
    return axios.get(`${baseCountUrl}?${params}`).then((response) => response.data.count);
}

/**
 * Fetches a specified alert, using the input hooks to give the results.
 *
 * @param {!string} id
 * @returns {Promise<Object, Error>} fulfilled with normalized response
 */
export function fetchAlert(id) {
    if (!id) {
        throw new Error('Alert ID must be specified');
    }
    return axios.get(`${baseUrl}/${id}`).then((response) => response.data);
}

/**
 * Resolves an alert given an alert ID and returns results through input functions.
 *
 * @param {!string} alertId
 * @param {bool} addToBaseline
 * @returns {Promise<AxiosResponse, Error>}
 */
export function resolveAlert(alertId, addToBaseline = false) {
    return axios.patch(`${baseUrl}/${alertId}/resolve`, { addToBaseline });
}

/**
 * Resolves a list of alerts by alert ID and returns results through input functions.
 *
 * @param {string[]} alertIds
 * @param {bool} addToBaseline
 * @returns {Promise<AxiosResponse, Error>}
 */
export function resolveAlerts(alertIds = [], addToBaseline = false) {
    return Promise.all(alertIds.map((id) => resolveAlert(id, addToBaseline)));
}
