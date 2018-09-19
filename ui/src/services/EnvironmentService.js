import axios from 'axios';
import queryString from 'query-string';

const baseUrl = '/v1/networkgraph';
const networkPoliciesUrl = '/v1/networkpolicies';

/**
 * Fetches nodes and links for the environment graph.
 * Returns response with nodes and links
 *
 * @returns {Promise<Object, Error>}
 */
export function fetchEnvironmentGraph(filters, clusterId) {
    const { query, simulationYaml } = filters;
    const params = queryString.stringify({ query });
    const options = {
        method: 'POST',
        data: simulationYaml && `"${simulationYaml.split('\n').join('\\n')}"`,
        url: `${baseUrl}/cluster/${clusterId}?${params}`
    };
    return axios(options).then(response => ({
        response: response.data
    }));
}

/**
 * Fetches policies details for given array of ids.
 *
 * @param {!array} policyIds
 * @returns {Promise<Object, Error>}
 */
export function fetchNetworkPolicies(policyIds) {
    const networkPoliciesPromises = policyIds.map(policyId =>
        axios.get(`${networkPoliciesUrl}/${policyId}`)
    );
    return axios
        .all([...networkPoliciesPromises])
        .then(response => ({ response: response.map(networkPolicy => networkPolicy.data) }));
}

/**
 * Fetches Node updates.
 *
 * @returns {Promise<Object, Error>}
 */
export function fetchNodeUpdates() {
    return axios.get(`${baseUrl}/epoch`).then(response => ({
        response: response.data
    }));
}

/**
 * Sends a notification of the simulated yaml
 *
 * @param {!String} clusterId
 * @param {!String} notifierId
 * @param {!String} simulationYaml
 * @returns {Promise<Object, Error>}
 */
export function sendYAMLNotification(clusterId, notifierId, simulationYaml) {
    const options = {
        method: 'POST',
        data: simulationYaml && `"${simulationYaml.split('\n').join('\\n')}"`,
        url: `${networkPoliciesUrl}/simulation/notify?cluster_id=${clusterId}&notifier_id=${notifierId}`
    };
    return axios(options).then(response => ({
        response: response.data
    }));
}
