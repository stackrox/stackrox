import axios from 'axios';
import { normalize } from 'normalizr';

import { policy as policySchema } from './schemas';

const baseUrl = '/v1/policies';

/**
 * Fetches policy details for a given policy ID.
 * Returns normalized response with policy entity extracted.
 *
 * @param {!string} policyId
 * @returns {Promise<Object, Error>} fulfilled with normalized response
 */
export function fetchPolicy(policyId) {
    return axios.get(`${baseUrl}/${policyId}`).then(response => ({
        response: normalize(response.data, policySchema)
    }));
}
/**
 * Updates policy with a given ID to add deployment into the whitelisted entries.
 *
 * @param {!string} policyId
 * @param {!string} deploymentName
 * @returns {Promise<AxiosResponse, Error>} fulfilled in case of success or rejected with an error
 */
export async function whitelistDeployment(policyId, deploymentName) {
    const { response } = await fetchPolicy(policyId);
    const policy = response.entities.policy[policyId];

    const deploymentEntry = {
        deployment: { name: deploymentName }
    };
    policy.whitelists = [...policy.whitelists, deploymentEntry];
    return axios.put(`${baseUrl}/${policy.id}`, policy);
}
