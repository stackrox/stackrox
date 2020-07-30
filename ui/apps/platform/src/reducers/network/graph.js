import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';
import set from 'lodash/set';
import get from 'lodash/get';
import { LOCATION_CHANGE } from 'react-router-redux';

import { types as backendTypes } from 'reducers/network/backend';
import { types as searchTypes } from 'reducers/network/search';

import { filterModes } from 'Containers/Network/Graph/filterModes';
import { isNonIsolatedNode } from 'utils/networkGraphUtils';
import entityTypes from 'constants/entityTypes';

export const networkGraphClusters = {
    KUBERNETES_CLUSTER: true,
    OPENSHIFT_CLUSTER: true,
};

let networkFlowGraphEnabled = false;

const setEdgeMapState = (graph, state, property) => {
    const newState = { ...state };
    graph.nodes.forEach((node) => {
        if (node?.entity?.type !== entityTypes.DEPLOYMENT) {
            return;
        }
        const { id: srcDeploymentId } = node.entity;
        Object.keys(node.outEdges).forEach((tgtIndex) => {
            const tgtNode = graph.nodes[tgtIndex];
            if (tgtNode?.entity?.type !== entityTypes.DEPLOYMENT) {
                return;
            }
            const { id: tgtDeploymentId } = tgtNode.entity;
            const mapKey = [srcDeploymentId, tgtDeploymentId].sort().join('--');
            if (!newState[mapKey]) newState[mapKey] = {};
            if (!newState[mapKey][property]) {
                newState[mapKey][property] = [];
            }
            newState[mapKey][property].push({
                source: srcDeploymentId,
                target: tgtDeploymentId,
            });
        });
    });
    return newState;
};

const setNodeMapState = (graph, state, propertyConfig) => {
    const newState = { ...state };
    const { ingressKey, egressKey } = propertyConfig;
    graph.nodes.forEach((node) => {
        if (node?.entity?.type !== entityTypes.DEPLOYMENT) {
            return;
        }
        const { id } = node.entity;
        if (!newState[id]) newState[id] = {};
        if (isNonIsolatedNode(node)) newState[id].nonIsolated = true;
        newState[id][egressKey] = [];
        Object.keys(node.outEdges).forEach((tgtIndex) => {
            const tgtNode = graph.nodes[tgtIndex];
            if (tgtNode?.entity?.type !== entityTypes.DEPLOYMENT) {
                return;
            }
            const { id: tgtDeploymentId } = tgtNode.entity;
            newState[id][egressKey].push(tgtDeploymentId);
            if (get(newState, [tgtDeploymentId, ingressKey])) {
                newState[tgtDeploymentId][ingressKey].push(id);
            } else {
                set(newState, [tgtDeploymentId, ingressKey], [id]);
            }
        });
    });
    return newState;
};

// Action types

export const types = {
    SET_NETWORK_GRAPH_REF: 'network/SET_NETWORK_GRAPH_REF',
    SET_NETWORK_GRAPH_FILTER_MODE: 'network/SET_NETWORK_GRAPH_FILTER_MODE',
    SET_NETWORK_EDGE_MAP: 'network/SET_NETWORK_EDGE_MAP',
    SET_NETWORK_NODE_MAP: 'network/SET_NETWORK_NODE_MAP',
    SET_SELECTED_NODE: 'network/SET_SELECTED_NODE',
    SET_SELECTED_NAMESPACE: 'network/SET_SELECTED_NAMESPACE',
    SELECT_DEFAULT_NETWORK_CLUSTER_ID: 'network/SELECT_DEFAULT_NETWORK_CLUSTER_ID',
    SELECT_NETWORK_CLUSTER_ID: 'network/SELECT_NETWORK_CLUSTER_ID',
    UPDATE_NETWORK_GRAPH_TIMESTAMP: 'network/UPDATE_NETWORK_GRAPH_TIMESTAMP',
    NETWORK_NODES_UPDATE: 'network/NETWORK_NODES_UPDATE',
    NETWORK_GRAPH_LOADING: 'network/NETWORK_GRAPH_LOADING',
};

// Actions

export const actions = {
    setNetworkGraphRef: (networkGraphRef) => ({
        type: types.SET_NETWORK_GRAPH_REF,
        networkGraphRef,
    }),
    setNetworkGraphFilterMode: (mode) => ({
        type: types.SET_NETWORK_GRAPH_FILTER_MODE,
        mode,
    }),
    setNetworkEdgeMap: (flowGraph, policyGraph) => ({
        type: types.SET_NETWORK_EDGE_MAP,
        flowGraph,
        policyGraph,
    }),
    setNetworkNodeMap: (flowGraph, policyGraph) => ({
        type: types.SET_NETWORK_NODE_MAP,
        flowGraph,
        policyGraph,
    }),
    setSelectedNode: (node) => ({ type: types.SET_SELECTED_NODE, node }),
    setSelectedNamespace: (namespace) => ({ type: types.SET_SELECTED_NAMESPACE, namespace }),
    selectDefaultNetworkClusterId: (response) => ({
        type: types.SELECT_DEFAULT_NETWORK_CLUSTER_ID,
        response,
    }),
    selectNetworkClusterId: (clusterId) => ({
        type: types.SELECT_NETWORK_CLUSTER_ID,
        clusterId,
    }),
    updateNetworkGraphTimestamp: (lastUpdatedTimestamp) => ({
        type: types.UPDATE_NETWORK_GRAPH_TIMESTAMP,
        lastUpdatedTimestamp,
    }),
    networkNodesUpdate: () => ({
        type: types.NETWORK_NODES_UPDATE,
    }),
    setNetworkGraphLoading: (isNetworkGraphLoading) => ({
        type: types.NETWORK_GRAPH_LOADING,
        isNetworkGraphLoading,
    }),
};

// Reducers
const networkGraphRef = (state = null, action) => {
    if (action.type === types.SET_NETWORK_GRAPH_REF) {
        return action.networkGraphRef;
    }
    return state;
};

const networkGraphFilterMode = (state = filterModes.active, action) => {
    if (action.type === types.SET_NETWORK_GRAPH_FILTER_MODE) {
        return action.mode;
    }
    return state;
};

const networkEdgeMap = (state = null, action) => {
    if (action.type === types.SET_NETWORK_EDGE_MAP) {
        const { flowGraph, policyGraph } = action;
        const allGraph = { ...flowGraph, ...policyGraph };
        const mappingEquals = isEqual(allGraph, state);
        if (mappingEquals) {
            return state;
        }
        let newState = {};
        newState = setEdgeMapState(flowGraph, newState, 'active');
        newState = setEdgeMapState(policyGraph, newState, 'allowed');
        return newState;
    }
    return state;
};

const networkNodeMap = (state = {}, action) => {
    if (action.type === types.SET_NETWORK_NODE_MAP) {
        const { flowGraph, policyGraph } = action;
        let newState = {};
        newState = setNodeMapState(flowGraph, newState, {
            ingressKey: 'ingressActive',
            egressKey: 'egressActive',
        });
        newState = setNodeMapState(policyGraph, newState, {
            ingressKey: 'ingressAllowed',
            egressKey: 'egressAllowed',
        });
        return newState;
    }
    return state;
};

const selectedNode = (state = null, action) => {
    if (action.type === types.SET_SELECTED_NODE) {
        return action.node;
    }
    return state;
};

const selectedNamespace = (state = null, action) => {
    if (action.type === types.SET_SELECTED_NAMESPACE) {
        return action.namespace;
    }
    return state;
};

const selectedNetworkClusterId = (state = null, action) => {
    if (!state && action.type === types.SELECT_DEFAULT_NETWORK_CLUSTER_ID) {
        const { cluster } = action.response.entities;
        const filteredClusters = Object.values(cluster).filter((c) => networkGraphClusters[c.type]);
        if (filteredClusters && filteredClusters.length) {
            const clusterId = filteredClusters[0].id;
            return isEqual(clusterId, state) ? state : clusterId;
        }
    }
    if (action.type === types.SELECT_NETWORK_CLUSTER_ID) {
        const { clusterId } = action;
        return isEqual(clusterId, state) ? state : clusterId;
    }
    return state;
};

const networkFlowGraphUpdateKey = (state = { key: 0 }, action) => {
    const { type, payload, options } = action;

    if (type === LOCATION_CHANGE && payload.pathname.startsWith('/main/network')) {
        return { key: state.key + 1 };
    }
    if (type === searchTypes.SET_SEARCH_OPTIONS) {
        const { length } = options;
        if (!length) return { key: state.key + 1 };
        if (length && !action.options[length - 1].type) return { key: state.key };
    }
    if (
        type === backendTypes.FETCH_NETWORK_POLICY_GRAPH.SUCCESS ||
        type === backendTypes.FETCH_NETWORK_FLOW_GRAPH.SUCCESS
    ) {
        if (type === backendTypes.FETCH_NETWORK_FLOW_GRAPH.SUCCESS) {
            networkFlowGraphEnabled = true;
            return { key: state.key + 1 };
        }
        if (type === backendTypes.FETCH_NETWORK_POLICY_GRAPH.SUCCESS && !networkFlowGraphEnabled) {
            return { key: state.key + 1 };
        }
    }
    return state;
};

const lastUpdatedTimestamp = (state = null, action) => {
    if (action.type === types.UPDATE_NETWORK_GRAPH_TIMESTAMP) {
        return action.lastUpdatedTimestamp;
    }
    return state;
};

const isNetworkGraphLoading = (state = true, action) => {
    if (action.type === types.NETWORK_GRAPH_LOADING) {
        return action.isNetworkGraphLoading;
    }
    return state;
};

const reducer = combineReducers({
    networkGraphRef,
    networkGraphFilterMode,
    networkEdgeMap,
    networkNodeMap,
    selectedNode,
    selectedNamespace,
    selectedNetworkClusterId,
    networkFlowGraphUpdateKey,
    lastUpdatedTimestamp,
    isNetworkGraphLoading,
});

// Selectors

const getNetworkGraphRef = (state) => state.networkGraphRef;
const getNetworkGraphFilterMode = (state) => state.networkGraphFilterMode;
const getNetworkEdgeMap = (state) => state.networkEdgeMap;
const getNetworkNodeMap = (state) => state.networkNodeMap;
const getSelectedNode = (state) => state.selectedNode;
const getSelectedNamespace = (state) => state.selectedNamespace;
const getSelectedNetworkClusterId = (state) => state.selectedNetworkClusterId;
const getNetworkFlowGraphUpdateKey = (state) => state.networkFlowGraphUpdateKey.key;
const getLastUpdatedTimestamp = (state) => state.lastUpdatedTimestamp;
const getNetworkGraphLoading = (state) => state.isNetworkGraphLoading;

export const selectors = {
    getNetworkGraphRef,
    getNetworkGraphFilterMode,
    getNetworkEdgeMap,
    getNetworkNodeMap,
    getSelectedNode,
    getSelectedNamespace,
    getSelectedNetworkClusterId,
    getNetworkFlowGraphUpdateKey,
    getLastUpdatedTimestamp,
    getNetworkGraphLoading,
};

export default reducer;
