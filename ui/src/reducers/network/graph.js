import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';
import { LOCATION_CHANGE } from 'react-router-redux';

import { types as backendTypes } from 'reducers/network/backend';
import { types as searchTypes } from 'reducers/network/search';

import filterModes from 'Containers/Network/Graph/filterModes';

export const networkGraphClusters = {
    KUBERNETES_CLUSTER: true,
    OPENSHIFT_CLUSTER: true
};

// Action types

export const types = {
    SET_NETWORK_GRAPH_FILTER_MODE: 'network/SET_NETWORK_GRAPH_FILTER_MODE',
    SET_NETWORK_FLOW_MAPPING: 'network/SET_NETWORK_FLOW_MAPPING',
    SET_SELECTED_NODE_ID: 'network/SET_SELECTED_NODE_ID',
    SELECT_DEFAULT_NETWORK_CLUSTER_ID: 'network/SELECT_DEFAULT_NETWORK_CLUSTER_ID',
    SELECT_NETWORK_CLUSTER_ID: 'network/SELECT_NETWORK_CLUSTER_ID',
    INCREMENT_NETWORK_GRAPH_UPDATE_KEY: 'network/INCREMENT_NETWORK_GRAPH_UPDATE_KEY',
    UPDATE_NETWORK_GRAPH_TIMESTAMP: 'network/UPDATE_NETWORK_GRAPH_TIMESTAMP',
    NETWORK_NODES_UPDATE: 'network/NETWORK_NODES_UPDATE'
};

// Actions

export const actions = {
    setNetworkGraphFilterMode: mode => ({
        type: types.SET_NETWORK_GRAPH_FILTER_MODE,
        mode
    }),
    setNetworkFlowMapping: flowGraph => ({
        type: types.SET_NETWORK_FLOW_MAPPING,
        flowGraph
    }),
    setSelectedNodeId: id => ({ type: types.SET_SELECTED_NODE_ID, id }),
    selectDefaultNetworkClusterId: response => ({
        type: types.SELECT_DEFAULT_NETWORK_CLUSTER_ID,
        response
    }),
    selectNetworkClusterId: clusterId => ({
        type: types.SELECT_NETWORK_CLUSTER_ID,
        clusterId
    }),
    incrementNetworkGraphUpdateKey: () => ({
        type: types.INCREMENT_NETWORK_GRAPH_UPDATE_KEY
    }),
    updateNetworkGraphTimestamp: lastUpdatedTimestamp => ({
        type: types.UPDATE_NETWORK_GRAPH_TIMESTAMP,
        lastUpdatedTimestamp
    }),
    networkNodesUpdate: () => ({
        type: types.NETWORK_NODES_UPDATE
    })
};

// Reducers

const networkGraphFilterMode = (state = filterModes.active, action) => {
    if (action.type === types.SET_NETWORK_GRAPH_FILTER_MODE) {
        return action.mode;
    }
    return state;
};

const networkFlowMapping = (state = {}, action) => {
    if (action.type === types.SET_NETWORK_FLOW_MAPPING) {
        const { flowGraph } = action;
        const flowEquals = isEqual(flowGraph, state);
        if (flowEquals) {
            return state;
        }
        const newState = Object.assign({}, state);
        flowGraph.nodes.forEach(node => {
            if (!node.entity || node.entity.type !== 'DEPLOYMENT') {
                return;
            }
            const { id: srcDeploymentId } = node.entity;
            Object.keys(node.outEdges).forEach(tgtIndex => {
                const tgtNode = flowGraph.nodes[tgtIndex];
                if (!tgtNode.entity || tgtNode.entity.type !== 'DEPLOYMENT') {
                    return;
                }
                const { id: tgtDeploymentId } = tgtNode.entity;
                newState[`${srcDeploymentId}--${tgtDeploymentId}`] = true;
            });
        });
        return newState;
    }
    return state;
};

const selectedNodeId = (state = null, action) => {
    if (action.type === types.SET_SELECTED_NODE_ID) {
        return action.id;
    }
    return state;
};

const selectedNetworkClusterId = (state = null, action) => {
    if (!state && action.type === types.SELECT_DEFAULT_NETWORK_CLUSTER_ID) {
        const { cluster } = action.response.entities;
        const filteredClusters = Object.values(cluster).filter(c => networkGraphClusters[c.type]);
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

const networkFlowGraphUpdateKey = (state = { shouldUpdate: true, key: 0 }, action) => {
    const { type, payload, options } = action;

    if (type === LOCATION_CHANGE && payload.pathname.startsWith('/main/network')) {
        return { shouldUpdate: true, key: state.key + 1 };
    }
    if (type === searchTypes.SET_SEARCH_OPTIONS) {
        const { length } = options;
        if (!length) return { shouldUpdate: true, key: state.key + 1 };
        if (length && !action.options[length - 1].type)
            return { shouldUpdate: true, key: state.key };
    }
    if (type === types.SELECT_NETWORK_CLUSTER_ID) {
        return { shouldUpdate: true, key: state.key + 1 };
    }
    if (type === types.INCREMENT_NETWORK_GRAPH_UPDATE_KEY) {
        return { shouldUpdate: true, key: state.key + 1 };
    }
    if (
        type === backendTypes.FETCH_NETWORK_POLICY_GRAPH.SUCCESS ||
        type === backendTypes.FETCH_NETWORK_FLOW_GRAPH.SUCCESS
    ) {
        if (state.shouldUpdate) return { shouldUpdate: false, key: state.key + 1 };
    }
    if (type === types.SET_NETWORK_FLOW_MAPPING) {
        return { shouldUpdate: true, key: state.key + 1 };
    }
    return state;
};

const lastUpdatedTimestamp = (state = null, action) => {
    if (action.type === types.UPDATE_NETWORK_GRAPH_TIMESTAMP) {
        return action.lastUpdatedTimestamp;
    }
    return state;
};

const reducer = combineReducers({
    networkGraphFilterMode,
    networkFlowMapping,
    selectedNodeId,
    selectedNetworkClusterId,
    networkFlowGraphUpdateKey,
    lastUpdatedTimestamp
});

// Selectors

const getNetworkGraphFilterMode = state => state.networkGraphFilterMode;
const getNetworkFlowMapping = state => state.networkFlowMapping;
const getSelectedNodeId = state => state.selectedNodeId;
const getSelectedNetworkClusterId = state => state.selectedNetworkClusterId;
const getNetworkFlowGraphUpdateKey = state => state.networkFlowGraphUpdateKey.key;
const getLastUpdatedTimestamp = state => state.lastUpdatedTimestamp;

export const selectors = {
    getNetworkGraphFilterMode,
    getNetworkFlowMapping,
    getSelectedNodeId,
    getSelectedNetworkClusterId,
    getNetworkFlowGraphUpdateKey,
    getLastUpdatedTimestamp
};

export default reducer;
