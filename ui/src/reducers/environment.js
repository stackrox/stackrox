import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

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
    SET_SELECTED_NODE_ID: { type: 'environment/SET_SELECTED_NODE_ID' },
    ...searchTypes('environment')
};

// Actions

export const actions = {
    fetchEnvironmentGraph: createFetchingActions(types.FETCH_ENVIRONMENT_GRAPH),
    fetchNetworkPolicies: createFetchingActions(types.FETCH_NETWORK_POLICIES),
    setSelectedNodeId: id => ({ type: types.SET_SELECTED_NODE_ID, id }),
    ...getSearchActions('environment')
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

const reducer = combineReducers({
    environmentGraph,
    deployment,
    networkPolicies,
    selectedNodeId,
    ...searchReducers('environment')
});

// Selectors

const getEnvironmentGraph = state => state.environmentGraph;
const getDeployment = state => state.deployment;
const getNetworkPolicies = state => state.networkPolicies;
const getSelectedNodeId = state => state.selectedNodeId;

export const selectors = {
    getEnvironmentGraph,
    getDeployment,
    getNetworkPolicies,
    getSelectedNodeId,
    ...getSearchSelectors('environment')
};

export default reducer;
