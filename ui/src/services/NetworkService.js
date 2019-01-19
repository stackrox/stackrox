import axios from 'axios';
import queryString from 'qs';

const networkPoliciesBaseUrl = '/v1/networkpolicies';
const networkFlowBaseUrl = '/v1/networkgraph';

/**
 * Fetches nodes and links for the network graph.
 * Returns response with nodes and links
 *
 * @returns {Promise<Object, Error>}
 */
export function fetchNetworkPolicyGraph(filters, clusterId) {
    const { query, simulationYaml } = filters;
    const params = queryString.stringify({ query }, { encode: false, arrayFormat: 'repeat' });
    let options;
    let getGraph = data => data;
    if (simulationYaml) {
        options = {
            method: 'POST',
            data: {
                applyYaml: simulationYaml
            },
            url: `${networkPoliciesBaseUrl}/simulate/${clusterId}?${params}`
        };
        getGraph = ({ simulatedGraph }) => simulatedGraph;
    } else {
        options = {
            method: 'GET',
            url: `${networkPoliciesBaseUrl}/cluster/${clusterId}?${params}`
        };
    }

    return axios(options).then(response => ({
        response: getGraph(response.data)
    }));
}

/**
 * Fetches nodes and links for the network flow graph.
 * Returns response with nodes and links
 *
 * @returns {Promise<Object, Error>}
 */
export function fetchNetworkFlowGraph(filters, clusterId) {
    const { query } = filters;
    const params = queryString.stringify({ query }, { encode: false, arrayFormat: 'repeat' });
    const options = {
        method: 'GET',
        url: `${networkFlowBaseUrl}/cluster/${clusterId}?${params}`
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
        axios.get(`${networkPoliciesBaseUrl}/${policyId}`)
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
    return axios.get(`${networkPoliciesBaseUrl}/graph/epoch`).then(response => ({
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
        url: `${networkPoliciesBaseUrl}/simulate/${clusterId}/notify?cluster_id=${clusterId}&notifier_id=${notifierId}`
    };
    return axios(options).then(response => ({
        response: response.data
    }));
}
