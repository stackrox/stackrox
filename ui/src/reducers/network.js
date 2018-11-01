import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';
import { LOCATION_CHANGE } from 'react-router-redux';

import { types as clusterTypes } from 'reducers/clusters';
import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';
import { types as deploymentTypes } from 'reducers/deployments';
import {
    types as searchTypes,
    getActions as getSearchActions,
    reducers as searchReducers,
    getSelectors as getSearchSelectors
} from 'reducers/pageSearch';

// Action types

export const types = {
    FETCH_NETWORK_POLICY_GRAPH: createFetchingActionTypes('network/FETCH_NETWORK_POLICY_GRAPH'),
    FETCH_NETWORK_FLOW_GRAPH: createFetchingActionTypes('network/FETCH_NETWORK_FLOW_GRAPH'),
    FETCH_NETWORK_POLICIES: createFetchingActionTypes('network/FETCH_NETWORK_POLICIES'),
    FETCH_NODE_UPDATES: createFetchingActionTypes('network/FETCH_NODE_UPDATES'),
    SET_NETWORK_FLOW_MAPPING: 'network/SET_NETWORK_FLOW_MAPPING',
    SET_SELECTED_NODE_ID: 'network/SET_SELECTED_NODE_ID',
    SELECT_NETWORK_CLUSTER_ID: 'network/SELECT_NETWORK_CLUSTER_ID',
    INCREMENT_NETWORK_GRAPH_UPDATE_KEY: 'network/INCREMENT_NETWORK_GRAPH_UPDATE_KEY',
    SIMULATOR_MODE_ON: 'network/SIMULATOR_MODE_ON',
    SET_NETWORK_GRAPH_STATE: 'network/NETWORK_GRAPH_STATE',
    SET_YAML_FILE: 'network/SET_YAML_FILE',
    SEND_YAML_NOTIFICATION: 'network/SEND_YAML_NOTIFICATION',
    UPDATE_NETWORKGRAPH_TIMESTAMP: 'network/UPDATE_NETWORKGRAPH_TIMESTAMP',
    NETWORK_NODES_UPDATE: 'network/NETWORK_NODES_UPDATE',
    ...searchTypes('network')
};

// Actions

// Network search should not show the 'Cluster' category
const getNetworkSearchActions = getSearchActions('network');
const networkSearchActions = Object.assign({}, getNetworkSearchActions);
const filterSearchOptions = options => options.filter(obj => obj.value !== 'Cluster:');
networkSearchActions.setNetworkSearchModifiers = options =>
    getNetworkSearchActions.setNetworkSearchModifiers(filterSearchOptions(options));
networkSearchActions.setNetworkSearchSuggestions = options =>
    getNetworkSearchActions.setNetworkSearchSuggestions(filterSearchOptions(options));

export const actions = {
    fetchNetworkPolicyGraph: createFetchingActions(types.FETCH_NETWORK_POLICY_GRAPH),
    fetchNetworkFlowGraph: createFetchingActions(types.FETCH_NETWORK_FLOW_GRAPH),
    fetchNetworkPolicies: createFetchingActions(types.FETCH_NETWORK_POLICIES),
    fetchNodeUpdates: createFetchingActions(types.FETCH_NODE_UPDATES),
    setNetworkFlowMapping: flowGraph => ({
        type: types.SET_NETWORK_FLOW_MAPPING,
        flowGraph
    }),
    setSelectedNodeId: id => ({ type: types.SET_SELECTED_NODE_ID, id }),
    selectNetworkClusterId: clusterId => ({
        type: types.SELECT_NETWORK_CLUSTER_ID,
        clusterId
    }),
    incrementNetworkGraphUpdateKey: () => ({
        type: types.INCREMENT_NETWORK_GRAPH_UPDATE_KEY
    }),
    networkNodesUpdate: () => ({
        type: types.NETWORK_NODES_UPDATE
    }),
    setNetworkGraphState: () => ({ type: types.SET_NETWORK_GRAPH_STATE }),
    setSimulatorMode: value => ({ type: types.SIMULATOR_MODE_ON, value }),
    setYamlFile: file => ({ type: types.SET_YAML_FILE, file }),
    sendYAMLNotification: notifierId => ({
        type: types.SEND_YAML_NOTIFICATION,
        notifierId
    }),
    updateNetworkGraphTimestamp: lastUpdatedTimestamp => ({
        type: types.UPDATE_NETWORKGRAPH_TIMESTAMP,
        lastUpdatedTimestamp
    }),
    ...networkSearchActions
};

// Reducers

const networkPolicyGraph = (state = { nodes: [] }, action) => {
    if (action.type === types.FETCH_NETWORK_POLICY_GRAPH.SUCCESS) {
        return isEqual(action.response, state) ? state : action.response;
    }
    return state;
};

const networkFlowGraph = (state = { nodes: [], edges: [] }, action) => {
    if (action.type === types.FETCH_NETWORK_FLOW_GRAPH.SUCCESS) {
        return isEqual(action.response, state) ? state : action.response;
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
            Object.keys(node.outEdges).forEach(tgtIndex => {
                const tgtNode = flowGraph.nodes[tgtIndex];
                newState[`${node.deploymentId}--${tgtNode.deploymentId}`] = true;
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

const selectedYamlFile = (state = null, action) => {
    if (action.type === types.SET_YAML_FILE) {
        return action.file;
    }
    return state;
};

const deployment = (state = {}, action) => {
    if (action.response && action.response.entities) {
        const { entities, result } = action.response;
        if (entities && entities.deployment && result) {
            const deploymentsById = Object.assign({}, entities.deployment[result]);
            if (action.type === deploymentTypes.FETCH_DEPLOYMENT.SUCCESS) {
                return isEqual(deploymentsById, state) ? state : deploymentsById;
            }
        }
    }
    return state;
};

const networkPolicies = (state = [], action) => {
    if (action.type === types.FETCH_NETWORK_POLICIES.SUCCESS) {
        return isEqual(action.response, state) ? state : action.response;
    }
    return state;
};

const nodeUpdatesEpoch = (state = null, action) => {
    if (action.type === types.FETCH_NODE_UPDATES.SUCCESS) {
        return isEqual(action.response.epoch, state) ? state : action.response.epoch;
    }
    return state;
};

const lastUpdatedTimestamp = (state = null, action) => {
    if (action.type === types.UPDATE_NETWORKGRAPH_TIMESTAMP) {
        return action.lastUpdatedTimestamp;
    }
    return state;
};

export const networkGraphClusters = {
    KUBERNETES_CLUSTER: true,
    OPENSHIFT_CLUSTER: true
};

const selectedNetworkClusterId = (state = null, action) => {
    if (!state && action.type === clusterTypes.FETCH_CLUSTERS.SUCCESS) {
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

const networkGraphUpdateKey = (state = { shouldUpdate: true, key: 0 }, action) => {
    const { type, payload, options } = action;

    if (type === LOCATION_CHANGE && payload.pathname.startsWith('/main/network')) {
        return { shouldUpdate: true, key: state.key + 1 };
    }
    if (type === types.SET_SEARCH_OPTIONS) {
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
        type === types.FETCH_NETWORK_POLICY_GRAPH.SUCCESS ||
        type === types.FETCH_NETWORK_FLOW_GRAPH.SUCCESS
    ) {
        if (state.shouldUpdate) return { shouldUpdate: false, key: state.key + 1 };
    }
    if (type === types.SET_NETWORK_FLOW_MAPPING) {
        return { shouldUpdate: true, key: state.key + 1 };
    }
    return state;
};

const networkGraphState = (state = 'INITIAL', action) => {
    const { type } = action;
    if (type === types.SET_NETWORK_GRAPH_STATE) {
        return 'INITIAL';
    }
    if (type === types.FETCH_NETWORK_POLICY_GRAPH.REQUEST) {
        return 'REQUEST';
    }
    if (type === types.FETCH_NETWORK_POLICY_GRAPH.FAILURE) {
        return 'ERROR';
    }
    if (type === types.FETCH_NETWORK_POLICY_GRAPH.SUCCESS) {
        return 'SUCCESS';
    }
    return state;
};

const simulatorMode = (state = false, action) => {
    if (action.type === types.SIMULATOR_MODE_ON) {
        return action.value;
    }
    return state;
};

const errorMessage = (state = '', action) => {
    if (action.type === types.FETCH_NETWORK_POLICY_GRAPH.FAILURE) {
        const { message } = action.error.response.data;
        return message;
    }
    return state;
};

const reducer = combineReducers({
    networkPolicyGraph,
    networkFlowGraph,
    networkFlowMapping,
    deployment,
    networkPolicies,
    selectedNodeId,
    nodeUpdatesEpoch,
    selectedNetworkClusterId,
    networkGraphUpdateKey,
    networkGraphState,
    simulatorMode,
    selectedYamlFile,
    errorMessage,
    lastUpdatedTimestamp,
    ...searchReducers('network')
});

// Selectors

const getNetworkPolicyGraph = state => state.networkPolicyGraph;
const getNetworkFlowGraph = state => state.networkFlowGraph;
const getNetworkFlowMapping = state => state.networkFlowMapping;
const getDeployment = state => state.deployment;
const getNetworkPolicies = state => state.networkPolicies;
const getSelectedNodeId = state => state.selectedNodeId;
const getNodeUpdatesEpoch = state => state.nodeUpdatesEpoch;
const getSelectedNetworkClusterId = state => state.selectedNetworkClusterId;
const getNetworkGraphUpdateKey = state => state.networkGraphUpdateKey.key;
const getNetworkGraphState = state => state.networkGraphState;
const getSimulatorMode = state => state.simulatorMode;
const getNetworkGraphErrorMessage = state => state.errorMessage;
const getYamlFile = state => state.selectedYamlFile;
const getLastUpdatedTimestamp = state => state.lastUpdatedTimestamp;

export const selectors = {
    getNetworkPolicyGraph,
    getNetworkFlowGraph,
    getNetworkFlowMapping,
    getDeployment,
    getNetworkPolicies,
    getSelectedNodeId,
    getNodeUpdatesEpoch,
    getSelectedNetworkClusterId,
    getNetworkGraphUpdateKey,
    getNetworkGraphState,
    getSimulatorMode,
    getNetworkGraphErrorMessage,
    getYamlFile,
    getLastUpdatedTimestamp,
    ...getSearchSelectors('network')
};

export default reducer;
