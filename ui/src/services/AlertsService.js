import { normalize } from 'normalizr';
import axios from 'axios';
import queryString from 'query-string';

import {
    alert as alertSchema,
    alertNumsByPolicy as alertNumsByPolicySchema,
    alerts as alertsSchema
} from './schemas';

const baseUrl = '/v1/alerts';

/**
 * Fetches non-stale alert counts grouped by policy.
 * Returns normalized response with policy entities extracted.
 *
 * @param {Object} [filters={}] map of filters "filter -> value"
 * @returns {Promise<Object, Error>} fulfilled with normalized response
 */
export function fetchAlertNumsByPolicy(filters = {}) {
    const params = queryString.stringify({
        ...filters,
        stale: false
    });
    return axios.get(`${baseUrl}/groups?${params}`).then(response => ({
        response: normalize(response.data, alertNumsByPolicySchema)
    }));
}

/**
 * Fetches non-stale alerts for a given policy.
 * Returns normalized response with alert entities extracted.
 *
 * @param {!string} policyId
 * @returns {Promise<Object, Error>} fulfilled with normalized response
 */
export function fetchAlertsByPolicy(policyId) {
    const params = queryString.stringify({
        policyId,
        stale: false
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
    // TODO-ivan: fix alert API to start with baseUrl
    return axios.get(`/v1/alert/${alertId}`).then(response => ({
        response: normalize(response.data, alertSchema)
    }));
}
