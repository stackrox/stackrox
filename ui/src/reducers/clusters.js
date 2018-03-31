import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

// Action types

export const types = {
    FETCH_CLUSTERS: createFetchingActionTypes('clusters/FETCH_CLUSTERS'),
    SELECT_CLUSTER: 'clusters/SELECT_CLUSTER',
    SELECT_CLUSTER_TYPE: 'clusters/SELECT_CLUSTER_TYPE',
    EDIT_CLUSTER: 'clusters/EDIT_CLUSTER',
    SET_CREATED_CLUSTER_ID: 'clusters/SET_CREATED_CLUSTER_ID'
};

// Actions

export const actions = {
    fetchClusters: createFetchingActions(types.FETCH_CLUSTERS),
    selectCluster: clusterId => ({ type: types.SELECT_CLUSTER, clusterId }),
    selectClusterType: clusterType => ({ type: types.SELECT_CLUSTER_TYPE, clusterType }),
    editCluster: clusterId => ({ type: types.EDIT_CLUSTER, clusterId }),
    setCreatedClusterId: clusterId => ({ type: types.SET_CREATED_CLUSTER_ID, clusterId })
};

// Reducers

const clusters = (state = [], action) => {
    if (action.type === types.FETCH_CLUSTERS.SUCCESS) {
        return isEqual(action.response.clusters, state) ? state : action.response.clusters;
    }
    return state;
};

const selectedCluster = (state = null, action) => {
    if (action.type === types.SELECT_CLUSTER) {
        return action.clusterId || null;
    }
    if (state && action.type === types.FETCH_CLUSTERS.SUCCESS) {
        // received a new list of clusters and it doesn't contain selected cluster: unselect
        if (!action.response.clusters.map(cluster => cluster.id).includes(state)) return null;
    }
    return state;
};

const editingCluster = (state = null, action) => {
    if (action.type === types.SAVE_CLUSTER) {
        return null;
    }
    if (action.type === types.EDIT_CLUSTER) {
        if (action.clusterId === undefined) {
            return { new: true } || null;
        } else if (action.clusterId) {
            return { id: action.clusterId, new: false } || null;
        }
        return null;
    }
    if (state && action.type === types.FETCH_CLUSTERS.SUCCESS) {
        // received a new list of clusters and it doesn't contain the cluster that was being edited: unselect
        if (state.id && !action.response.clusters.map(cluster => cluster.id).includes(state.id)) {
            return null;
        }
    }
    return state;
};

const createdClusterId = (state = null, action) => {
    if (action.type === types.SET_CREATED_CLUSTER_ID) {
        return action.clusterId || null;
    }
    return state;
};

const selectedClusterType = (state = null, action) => {
    if (action.type === types.SELECT_CLUSTER_TYPE) {
        return action.clusterType || null;
    }
    return state;
};

const reducer = combineReducers({
    clusters,
    selectedCluster,
    editingCluster,
    selectedClusterType,
    createdClusterId
});

export default reducer;

// Selectors

const getClusters = state => state.clusters;
const getSelectedClusterId = state => state.selectedCluster;
const getEditingCluster = state => state.editingCluster;
const getSelectedClusterType = state => state.selectedClusterType;
const getCreatedClusterId = state => state.createdClusterId;

export const selectors = {
    getClusters,
    getSelectedClusterType,
    getSelectedClusterId,
    getEditingCluster,
    getCreatedClusterId
};
