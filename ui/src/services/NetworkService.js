import queryString from 'qs';
import axios from './instance';

const networkPoliciesBaseUrl = '/v1/networkpolicies';
const networkFlowBaseUrl = '/v1/networkgraph';

/**
 * Fetches nodes and links for the network graph.
 * Returns response with nodes and links
 *
 * @returns {Promise<Object, Error>}
 */
export function fetchNetworkPolicyGraph(clusterId, query, modification) {
    const params = queryString.stringify({ query }, { arrayFormat: 'repeat' });
    let options;
    let getGraph = data => data;
    if (modification) {
        options = {
            method: 'POST',
            data: modification,
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
export function fetchNetworkFlowGraph(clusterId, query, date) {
    let params;
    if (date) {
        const since = date.toISOString();
        params = queryString.stringify({ query, since }, { arrayFormat: 'repeat' });
    } else {
        params = queryString.stringify({ query }, { arrayFormat: 'repeat' });
    }
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
 * Generates a modification to policies based on a graph.
 *
 * @param {!String} clusterId
 * @param {!Object} query
 * @param {!String} since
 * @returns {Promise<Object, Error>}
 */
export function generateNetworkModification(clusterId, query, date) {
    let params;
    if (date) {
        const since = date.toISOString();
        params = queryString.stringify({ query, since }, { arrayFormat: 'repeat' });
    } else {
        params = queryString.stringify({ query }, { arrayFormat: 'repeat' });
    }
    const options = {
        method: 'GET',
        url: `${networkPoliciesBaseUrl}/generate/${clusterId}?deleteExisting=ALL&${params}`
    };
    return axios(options).then(response => response.data.modification);
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
 * @param {!array} notifierIds
 * @param {!Object} modification
 * @returns {Promise<Object, Error>}
 */
export function notifyNetworkPolicyModification(clusterId, notifierIds, modification) {
    const notifiers = queryString.stringify({ notifierIds }, { arrayFormat: 'repeat' });
    const options = {
        method: 'POST',
        data: modification,
        url: `${networkPoliciesBaseUrl}/simulate/${clusterId}/notify?${notifiers}`
    };
    return axios(options).then(response => ({
        response: response.data
    }));
}
