import { combineReducers } from 'redux';
import { createSelector } from 'reselect';
import isEqual from 'lodash/isEqual';
import pick from 'lodash/pick';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';
import mergeEntitiesById from 'utils/mergeEntitiesById';

export const clusterFormId = 'cluster-form';

export const clusterTypes = ['SWARM_CLUSTER', 'OPENSHIFT_CLUSTER', 'KUBERNETES_CLUSTER'];

export const wizardPages = Object.freeze({
    FORM: 'FORM',
    DEPLOYMENT: 'DEPLOYMENT'
});

// Action types

export const types = {
    FETCH_CLUSTERS: createFetchingActionTypes('clusters/FETCH_CLUSTERS'),
    FETCH_CLUSTER: createFetchingActionTypes('clusters/FETCH_CLUSTER'),
    SELECT_CLUSTER: 'clusters/SELECT_CLUSTER',
    START_WIZARD: 'clusters/START_WIZARD',
    NEXT_WIZARD_PAGE: 'clusters/NEXT_WIZARD_PAGE',
    PREV_WIZARD_PAGE: 'clusters/NEXT_WIZARD_PAGE',
    UPDATE_WIZARD_STATE: 'clusters/UPDATE_WIZARD_STATE',
    FINISH_WIZARD: 'clusters/FINISH_WIZARD',
    SAVE_CLUSTER: createFetchingActionTypes('clusters/SAVE_CLUSTER'),
    DELETE_CLUSTERS: 'clusters/DELETE_CLUSTERS',
    DOWNLOAD_CLUSTER_YAML: 'clusters/DOWNLOAD_CLUSTER_YAML'
};

// Actions

export const actions = {
    fetchClusters: createFetchingActions(types.FETCH_CLUSTERS),
    fetchCluster: createFetchingActions(types.FETCH_CLUSTER),
    selectCluster: clusterId => ({ type: types.SELECT_CLUSTER, clusterId }),
    startWizard: clusterId => ({ type: types.START_WIZARD, clusterId }),
    nextWizardPage: () => ({ type: types.NEXT_WIZARD_PAGE }),
    prevWizardPage: () => ({ type: types.PREV_WIZARD_PAGE }),
    updateWizardState: (page, clusterId) => ({ type: types.UPDATE_WIZARD_STATE, page, clusterId }),
    finishWizard: () => ({ type: types.FINISH_WIZARD }),
    saveCluster: createFetchingActions(types.SAVE_CLUSTER),
    deleteClusters: clusterIds => ({ type: types.DELETE_CLUSTERS, clusterIds }),
    downloadClusterYaml: clusterId => ({ type: types.DOWNLOAD_CLUSTER_YAML, clusterId })
};

// Reducers

const byId = (state = {}, { type, response }) => {
    if (type === types.FETCH_CLUSTERS.SUCCESS) {
        if (!response.entities.cluster) {
            return {};
        }
        const clustersById = response.entities.cluster;
        const newState = mergeEntitiesById(state, clustersById);
        const onlyExisting = pick(newState, Object.keys(clustersById));
        return isEqual(onlyExisting, state) ? state : onlyExisting;
    }
    if (type === types.SAVE_CLUSTER.SUCCESS || type === types.FETCH_CLUSTER.SUCCESS) {
        const clustersById = response.entities.cluster;
        return mergeEntitiesById(state, clustersById);
    }
    return state;
};

const selectedCluster = (state = null, action) => {
    if (action.type === types.SELECT_CLUSTER) {
        return action.clusterId;
    }
    if (state && action.type === types.FETCH_CLUSTERS.SUCCESS) {
        const clusters = action.response.entities.cluster;
        // received a new list of clusters and it doesn't contain selected cluster: unselect
        if (!clusters[state]) return null;
    }
    if (state && action.type === types.START_WIZARD) {
        // started add / edit wizard, deselect cluster if we're adding a new one
        return state === action.clusterId ? state : null;
    }
    return state;
};

const wizard = (state = null, { type, clusterId, page }) => {
    switch (type) {
        case types.START_WIZARD:
            return { page: wizardPages.FORM, clusterId };
        case types.UPDATE_WIZARD_STATE:
            return { page, clusterId };
        case types.FINISH_WIZARD:
            return null;
        default:
            return state;
    }
};

const reducer = combineReducers({
    byId,
    selectedCluster,
    wizard
});

export default reducer;

// Selectors

const getClustersById = state => state.byId;
const getClusters = createSelector([getClustersById], clusters => Object.values(clusters));
const getSelectedClusterId = state => state.selectedCluster;
const getWizardCurrentPage = state => (state.wizard ? state.wizard.page : null);
const getWizardClusterId = state => (state.wizard ? state.wizard.clusterId : null);

export const selectors = {
    getClustersById,
    getClusters,
    getSelectedClusterId,
    getWizardCurrentPage,
    getWizardClusterId
};
