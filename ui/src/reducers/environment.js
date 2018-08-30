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
    FETCH_ENVIRONMENT_GRAPH: createFetchingActionTypes('environment/FETCH_ENVIRONMENT_GRAPH'),
    FETCH_NETWORK_POLICIES: createFetchingActionTypes('environment/FETCH_NETWORK_POLICIES'),
    FETCH_NODE_UPDATES: createFetchingActionTypes('environment/FETCH_NODE_UPDATES'),
    SET_SELECTED_NODE_ID: { type: 'environment/SET_SELECTED_NODE_ID' },
    SELECT_ENVIRONMENT_CLUSTER_ID: 'environment/SELECT_ENVIRONMENT_CLUSTER_ID',
    INCREMENT_ENVIRONMENT_GRAPH_UPDATE_KEY: 'environment/INCREMENT_ENVIRONMENT_GRAPH_UPDATE_KEY',
    ...searchTypes('environment')
};

// Actions

// Environment search should not show the 'Cluster' category
const getEnvironmentSearchActions = getSearchActions('environment');
const environmentSearchActions = Object.assign({}, getEnvironmentSearchActions);
const filterSearchOptions = options => options.filter(obj => obj.value !== 'Cluster:');
environmentSearchActions.setEnvironmentSearchModifiers = options =>
    getEnvironmentSearchActions.setEnvironmentSearchModifiers(filterSearchOptions(options));
environmentSearchActions.setEnvironmentSearchSuggestions = options =>
    getEnvironmentSearchActions.setEnvironmentSearchSuggestions(filterSearchOptions(options));

export const actions = {
    fetchEnvironmentGraph: createFetchingActions(types.FETCH_ENVIRONMENT_GRAPH),
    fetchNetworkPolicies: createFetchingActions(types.FETCH_NETWORK_POLICIES),
    fetchNodeUpdates: createFetchingActions(types.FETCH_NODE_UPDATES),
    setSelectedNodeId: id => ({ type: types.SET_SELECTED_NODE_ID, id }),
    selectEnvironmentClusterId: clusterId => ({
        type: types.SELECT_ENVIRONMENT_CLUSTER_ID,
        clusterId
    }),
    incrementEnvironmentGraphUpdateKey: () => ({
        type: types.INCREMENT_ENVIRONMENT_GRAPH_UPDATE_KEY
    }),
    ...environmentSearchActions
};

// Reducers

const environmentGraph = (state = { nodes: [], edges: [] }, action) => {
    if (action.type === types.FETCH_ENVIRONMENT_GRAPH.SUCCESS) {
        return isEqual(action.response, state) ? state : action.response;
    }
    return state;
};

const selectedNodeId = (state = null, action) => {
    if (action.type === types.SET_SELECTED_NODE_ID) {
        return action.id;
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

export const networkGraphClusters = {
    KUBERNETES_CLUSTER: true,
    OPENSHIFT_CLUSTER: true
};

const selectedEnvironmentClusterId = (state = null, action) => {
    if (!state && action.type === clusterTypes.FETCH_CLUSTERS.SUCCESS) {
        const { cluster } = action.response.entities;
        const filteredClusters = Object.values(cluster).filter(c => networkGraphClusters[c.type]);
        if (filteredClusters && filteredClusters.length) {
            const clusterId = filteredClusters[0].id;
            return isEqual(clusterId, state) ? state : clusterId;
        }
    }
    if (action.type === types.SELECT_ENVIRONMENT_CLUSTER_ID) {
        const { clusterId } = action;
        return isEqual(clusterId, state) ? state : clusterId;
    }
    return state;
};

const environmentGraphUpdateKey = (state = { shouldUpdate: true, key: 0 }, action) => {
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
    if (type === types.SELECT_ENVIRONMENT_CLUSTER_ID) {
        return { shouldUpdate: true, key: state.key + 1 };
    }
    if (type === types.INCREMENT_ENVIRONMENT_GRAPH_UPDATE_KEY) {
        return { shouldUpdate: true, key: state.key + 1 };
    }
    if (type === types.FETCH_ENVIRONMENT_GRAPH.SUCCESS) {
        if (state.shouldUpdate) return { shouldUpdate: false, key: state.key + 1 };
    }
    return state;
};

const reducer = combineReducers({
    environmentGraph,
    deployment,
    networkPolicies,
    selectedNodeId,
    nodeUpdatesEpoch,
    selectedEnvironmentClusterId,
    environmentGraphUpdateKey,
    ...searchReducers('environment')
});

// Selectors

const getEnvironmentGraph = state => state.environmentGraph;
const getDeployment = state => state.deployment;
const getNetworkPolicies = state => state.networkPolicies;
const getSelectedNodeId = state => state.selectedNodeId;
const getNodeUpdatesEpoch = state => state.nodeUpdatesEpoch;
const getSelectedEnvironmentClusterId = state => state.selectedEnvironmentClusterId;
const getEnvironmentGraphUpdateKey = state => state.environmentGraphUpdateKey.key;

export const selectors = {
    getEnvironmentGraph,
    getDeployment,
    getNetworkPolicies,
    getSelectedNodeId,
    getNodeUpdatesEpoch,
    getSelectedEnvironmentClusterId,
    getEnvironmentGraphUpdateKey,
    ...getSearchSelectors('environment')
};

export default reducer;
