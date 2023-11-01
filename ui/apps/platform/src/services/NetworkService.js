import queryString from 'qs';

import { ORCHESTRATOR_COMPONENTS_KEY } from 'utils/orchestratorComponents';
import { convertToExactMatch } from 'utils/searchUtils';

import axios from './instance';

const networkPoliciesBaseUrl = '/v1/networkpolicies';
const networkFlowBaseUrl = '/v1/networkgraph';
const networkBaselineBaseUrl = '/v1/networkbaseline';

// for large clusters network graph requests may take time to process, so
// removing any global default timeout
const NETWORK_GRAPH_REQUESTS_TIMEOUT = 0;

/*
 * Applies the given network policy to the specified deployment
 *
 * @returns {Promise<Object, Error>}
 *
 */
export function applyBaselineNetworkPolicy({ deploymentId, modification }) {
    return axios
        .post(`${networkPoliciesBaseUrl}/apply/deployment/${deploymentId}`, {
            modification,
        })
        .then((response) => {
            return response.data;
        })
        .catch((error) => {
            return error.response.data;
        });
}

/*
 * Fetches the generated network policy modification for the specified deployment
 * from its network baseline
 *
 * @returns {Promise<Object, Error>}
 *
 */
export function fetchBaselineGeneratedNetworkPolicy({ deploymentId, includePorts }) {
    return axios
        .post(`${networkPoliciesBaseUrl}/generate/baseline/${deploymentId}`, {
            includePorts,
        })
        .then((response) => {
            return response.data;
        });
}

/*
 * Fetches the diff view of flows between the network policies currently applied to the
 * specified deployment and the baseline of that deployment.
 *
 * @returns {Promise<Object, Error>}
 *
 */
export function fetchBaselineComparison({ deploymentId }) {
    return axios
        .get(`${networkPoliciesBaseUrl}/baselinecomparison/${deploymentId}`)
        .then((response) => {
            return response.data;
        });
}

// TODO: wire this through redux saga, like the `fetchBaselineComparison` above
/*
 * Fetches the diff view of flows between the network policies last applied to the
 * specified deployment and the previous state before that application.
 *
 * @returns {Promise<Object, Error>}
 *
 */
export function fetchUndoComparison({ deploymentId }) {
    return axios
        .get(`${networkPoliciesBaseUrl}/undobaselinecomparison/${deploymentId}`)
        .then((response) => {
            return response.data;
        });
}

/*
 * Fetches the baselines status of the network flow
 *
 * @returns {Promise<Object, Error>}
 *
 */
export function fetchNetworkBaselineStatuses({ deploymentId, peers }) {
    return axios
        .post(`${networkBaselineBaseUrl}/${deploymentId}/status`, { peers })
        .then((response) => {
            return response.data;
        });
}

/*
 * Mark a flow (a connection of to a peer) as baselined or anomalous
 *
 * @returns {Promise<Object, Error>}
 *
 */
export function markNetworkBaselineStatuses({ deploymentId, networkBaselines }) {
    return axios
        .patch(`${networkBaselineBaseUrl}/${deploymentId}/peers`, { peers: networkBaselines })
        .then((response) => {
            return response.data;
        });
}

/*
 * Fetches the network baselines
 *
 * @returns {Promise<Object, Error>}
 *
 */
export function fetchNetworkBaselines({ deploymentId }) {
    return axios.get(`${networkBaselineBaseUrl}/${deploymentId}`).then((response) => {
        return response.data;
    });
}

/*
 * Enables or disables alerts on baseline violations
 *
 * @returns {Promise<Object, Error>}
 *
 */
export function toggleAlertBaselineViolations({ deploymentId, enable }) {
    const baseURL = `${networkBaselineBaseUrl}/${deploymentId}`;
    const URL = enable ? `${baseURL}/lock` : `${baseURL}/unlock`;
    return axios.patch(URL).then((response) => {
        return response.data;
    });
}

/*
 * Retrieves the last security policy applied for a deployment
 *
 * @param   {string}  deploymentId
 * @returns {Promise<Object, Error>}
 *
 */
export function getUndoModificationForDeployment(deploymentId) {
    const url = `${networkPoliciesBaseUrl}/undo/deployment/${deploymentId}`;
    return axios.get(url).then((response) => {
        return response.data;
    });
}

/**
 * Fetches nodes and links for the network graph.
 * Returns response with nodes and links
 *
 * @returns {Promise<Object, Error>}
 */
export function fetchNetworkPolicyGraph(
    clusterId,
    namespaces,
    deployments,
    query,
    modification,
    includePorts,
    includeOrchestratorComponents = false
) {
    const urlParams = query ? { query } : {};
    const namespaceQuery = namespaces.length > 0 ? `Namespace:${namespaces.join(',')}` : '';
    const deploymentQuery = deployments.length > 0 ? `Deployment:${deployments.join(',')}` : '';
    urlParams.query = query ? `${query}+${namespaceQuery}` : namespaceQuery;
    urlParams.query = deploymentQuery ? `${urlParams.query}+${deploymentQuery}` : urlParams.query;

    if (includePorts) {
        urlParams.includePorts = true;
    }

    // for openshift filtering toggle
    if (
        !includeOrchestratorComponents &&
        localStorage.getItem(ORCHESTRATOR_COMPONENTS_KEY) !== 'true'
    ) {
        urlParams.scope = {
            query: 'Orchestrator Component:false',
        };
    }
    const params = queryString.stringify(urlParams, { arrayFormat: 'repeat', allowDots: true });

    let options;
    let getGraph = (data) => data;
    if (modification) {
        options = {
            method: 'POST',
            data: modification,
            url: `${networkPoliciesBaseUrl}/simulate/${clusterId}?${params}`,
        };
        getGraph = ({ simulatedGraph }) => simulatedGraph;
    } else {
        options = {
            method: 'GET',
            url: `${networkPoliciesBaseUrl}/cluster/${clusterId}?${params}`,
        };
    }
    options = {
        ...options,
        timeout: NETWORK_GRAPH_REQUESTS_TIMEOUT,
    };
    return axios(options).then((response) => ({
        response: getGraph(response.data),
    }));
}

/**
 * Fetches nodes and links for the network flow graph.
 * Returns response with nodes and links
 *
 * @param {!String} clusterId
 * @param {String[]} namespaces
 * @param {String[]} deployments
 * @param {String} query
 * @param {Date} date
 * @param {boolean} includePorts
 * @param {boolean} includeOrchestratorComponents
 *
 * @returns {Promise<Object, Error>}
 */
export function fetchNetworkFlowGraph(
    clusterId,
    namespaces,
    deployments,
    query = '',
    date = null,
    includePorts = false,
    includeOrchestratorComponents = false,
    includePolicies = false
) {
    const urlParams = query ? { query } : {};
    const namespaceQuery =
        namespaces.length > 0 ? `Namespace:${namespaces.map(convertToExactMatch).join(',')}` : '';
    const deploymentQuery =
        deployments.length > 0
            ? `Deployment:${deployments.map(convertToExactMatch).join(',')}`
            : '';
    urlParams.query = query ? `${query}+${namespaceQuery}` : namespaceQuery;
    urlParams.query = deploymentQuery ? `${urlParams.query}+${deploymentQuery}` : urlParams.query;
    if (date) {
        urlParams.since = date.toISOString();
    }
    if (includePorts) {
        urlParams.includePorts = true;
    }
    if (includePolicies) {
        urlParams.include_policies = true;
    }
    // for openshift filtering toggle
    if (
        !includeOrchestratorComponents &&
        localStorage.getItem(ORCHESTRATOR_COMPONENTS_KEY) !== 'true'
    ) {
        urlParams.scope = {
            query: 'Orchestrator Component:false',
        };
    }
    const params = queryString.stringify(urlParams, { arrayFormat: 'repeat', allowDots: true });
    const options = {
        method: 'GET',
        url: `${networkFlowBaseUrl}/cluster/${clusterId}?${params}`,
        timeout: NETWORK_GRAPH_REQUESTS_TIMEOUT,
    };
    return axios(options).then((response) => ({
        response: response.data,
    }));
}

/**
 * Fetches policies details for given array of ids.
 *
 * @param {!String} namespaceId
 * @returns {Promise<Object, Error>}
 */
export function fetchNetworkPoliciesInNamespace(clusterId, namespaceId) {
    const options = {
        method: 'GET',
        url: `${networkPoliciesBaseUrl}?cluster_id=${clusterId}&namespace=${namespaceId}`,
    };
    return axios(options).then((response) => response.data.networkPolicies);
}

/**
 * Fetches policies details for given array of ids.
 *
 * @param {!array} policyIds
 * @returns {Promise<Object, Error>}
 */
export function fetchNetworkPolicies(policyIds) {
    const networkPoliciesPromises = policyIds.map((policyId) =>
        axios.get(`${networkPoliciesBaseUrl}/${policyId}`)
    );
    return Promise.allSettled(networkPoliciesPromises).then((responses) => {
        const responseData = {
            policies: [],
            errors: [],
        };

        responses.forEach((response) => {
            if (response.status === 'fulfilled') {
                responseData.policies.push(response.value.data);
            } else {
                responseData.errors.push(response.reason);
            }
        });
        return responseData;
    });
}

/**
 * Fetches Node updates.
 *
 * @returns {Promise<Object, Error>}
 */
export function fetchNodeUpdates(clusterId) {
    return axios
        .get(`${networkPoliciesBaseUrl}/graph/epoch?clusterId=${clusterId}`)
        .then((response) => ({
            response: response.data,
        }));
}

/**
 * Fetches the network policies currently applied to a cluster and set of deployments (defined by query).
 *
 * @param {!String} clusterId
 * @param {!Object} query
 * @returns {Promise<Object, Error>}
 */
export function getActiveNetworkModification(clusterId, deploymentQuery) {
    let params;
    if (deploymentQuery) {
        params = queryString.stringify({ clusterId, deploymentQuery }, { arrayFormat: 'repeat' });
    } else {
        params = queryString.stringify({ clusterId });
    }
    const options = {
        method: 'GET',
        url: `${networkPoliciesBaseUrl}?${params}`,
    };
    return axios(options).then((response) => {
        const policies = response?.data?.networkPolicies;
        if (policies) {
            return { applyYaml: policies.map((policy) => policy.yaml).join('\n---\n') };
        }
        return null;
    });
}

/**
 * Fetches the network policies applied to deployments in the given scope.
 * @param {!String} clusterId The cluster ID.
 * @param {!String} deploymentQuery A search filter string.
 * @returns {Promise<import("../types/networkPolicy.proto").NetworkPolicy[]>}
 */
export function fetchNetworkPoliciesByClusterId(clusterId, deploymentQuery) {
    if (clusterId === '') {
        return Promise.reject(new Error('A cluster ID must be provided to fetch network policies'));
    }
    // The `deploymentQuery` param functions identically to the general `query` param used in
    // other API calls and accepts the same search filter syntax.
    const params = queryString.stringify({ clusterId, deploymentQuery });
    const options = {
        method: 'GET',
        url: `${networkPoliciesBaseUrl}?${params}`,
    };
    return axios(options).then((response) => response.data.networkPolicies ?? []);
}

/**
 * Retrieves the modification that will undo the last action done through the stackrox UI.
 *
 * @param {!String} clusterId
 * @param {!Object} query
 * @returns {Promise<Object, Error>}
 */
export function getUndoNetworkModification(clusterId) {
    const options = {
        method: 'GET',
        url: `${networkPoliciesBaseUrl}/undo/${clusterId}`,
    };
    return axios(options).then((response) => response?.data?.undoRecord.undoModification ?? {});
}

/**
 * Generates a modification to policies based on a graph.
 *
 * @param {!String} clusterId
 * @param {!String} query
 * @param {!String} networkDataSince
 * @param {Boolean} excludePortsProtocols
 * @returns {Promise<Object, Error>}
 */
export function generateNetworkModification(
    clusterId,
    query,
    networkDataSince,
    excludePortsProtocols = null
) {
    const urlParams = query ? { query } : {};
    if (networkDataSince) {
        urlParams.networkDataSince = networkDataSince.toISOString();
    }

    if (excludePortsProtocols !== null) {
        urlParams.includePorts = !excludePortsProtocols;
    }

    const params = queryString.stringify(urlParams, { arrayFormat: 'repeat' });
    const options = {
        method: 'GET',
        url: `${networkPoliciesBaseUrl}/generate/${clusterId}?deleteExisting=NONE&${params}`,
    };
    return axios(options).then((response) => response?.data?.modification ?? {});
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
        url: `${networkPoliciesBaseUrl}/simulate/${clusterId}/notify?${notifiers}`,
    };
    return axios(options).then((response) => ({
        response: response.data,
    }));
}

/**
 * Sends a yaml to the backed for application to a cluster.
 *
 * @param {!String} clusterId
 * @param {!Object} modification
 * @returns {Promise<Object, Error>}
 */
export function applyNetworkPolicyModification(clusterId, modification) {
    const options = {
        method: 'POST',
        data: modification,
        url: `${networkPoliciesBaseUrl}/apply/${clusterId}`,
        timeout: NETWORK_GRAPH_REQUESTS_TIMEOUT,
    };
    return axios(options).then((response) => ({
        response: response.data,
    }));
}

/**
 * Fetches currently configured CIDR blocks.
 *
 * @returns {Promise<Object, Error>}
 */
export function fetchCIDRBlocks(clusterId) {
    // UI must always hide the default external sources.
    // TODO: Update this to search options pattern.
    const params = queryString.stringify(
        { query: 'Default External Source:false' },
        { arrayFormat: 'repeat', allowDots: true }
    );
    return axios
        .get(`${networkFlowBaseUrl}/cluster/${clusterId}/externalentities?${params}`)
        .then((response) => ({
            response: response.data,
        }));
}

/**
 * Posts a newly configured CIDR block.
 *
 * @returns {Promise<Object, Error>}
 */
export function postCIDRBlock(clusterId, block) {
    return axios
        .post(`${networkFlowBaseUrl}/cluster/${clusterId}/externalentities`, block)
        .then((response) => ({
            response: response.data,
        }));
}

/**
 * Patches an edited CIDR block name.
 *
 * @returns {Promise<Object, Error>}
 */
export function patchCIDRBlock(blockId, name) {
    return axios
        .patch(`${networkFlowBaseUrl}/externalentities/${blockId}`, { name })
        .then((response) => ({
            response: response.data,
        }));
}

/**
 * Deletes a previously configured CIDR block.
 *
 * @returns {Promise<Object, Error>}
 */
export function deleteCIDRBlock(blockId) {
    return axios.delete(`${networkFlowBaseUrl}/externalentities/${blockId}`).then((response) => ({
        response: response.data,
    }));
}

/**
 * Gets the default application generated CIDR blocks toggle state.
 *
 * @returns {Promise<Object, Error>}
 */
export function getHideDefaultExternalSrcs() {
    return axios.get(`${networkFlowBaseUrl}/config`).then((response) => ({
        response: response.data,
    }));
}

/**
 * Sets the default application generated CIDR blocks to be on or off.
 *
 * @returns {Promise<Object, Error>}
 */
export function setHideDefaultExternalSrcs(toggleState) {
    return axios
        .put(`${networkFlowBaseUrl}/config`, { config: { hideDefaultExternalSrcs: toggleState } })
        .then((response) => ({
            response: response.data,
        }));
}
