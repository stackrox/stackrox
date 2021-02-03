import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

export const clusterInitBundleFormId = 'cluster-init-bundle-form-id';

export const types = {
    FETCH_CLUSTER_INIT_BUNDLES: createFetchingActionTypes(
        'clusterInitBundles/FETCH_CLUSTER_INIT_BUNDLES'
    ),
    GENERATE_CLUSTER_INIT_BUNDLE: createFetchingActionTypes(
        'clusterInitBundles/GENERATE_CLUSTER_INIT_BUNDLE'
    ),
    // Note: REVOKE_CLUSTER_INIT_BUNDLES does not appear in reducers
    //       because FETCH_CLUSTER_INIT_BUNDLES is side-effect of saga, if revoke is successful.
    REVOKE_CLUSTER_INIT_BUNDLES: 'clusterInitBundles/REVOKE_CLUSTER_INIT_BUNDLES',
    START_CLUSTER_INIT_BUNDLE_GENERATION_WIZARD:
        'clusterInitBundles/START_CLUSTER_INIT_BUNDLE_GENERATION_WIZARD',
    CLOSE_CLUSTER_INIT_BUNDLE_GENERATION_WIZARD:
        'clusterInitBundles/CLOSE_CLUSTER_INIT_BUNDLE_GENERATION_WIZARD',
};

export const actions = {
    fetchClusterInitBundles: createFetchingActions(types.FETCH_CLUSTER_INIT_BUNDLES),
    generateClusterInitBundle: createFetchingActions(types.GENERATE_CLUSTER_INIT_BUNDLE),
    revokeClusterInitBundles: (ids) => ({ type: types.REVOKE_CLUSTER_INIT_BUNDLES, ids }),
    startClusterInitBundleGenerationWizard: () => ({
        type: types.START_CLUSTER_INIT_BUNDLE_GENERATION_WIZARD,
    }),
    closeClusterInitBundleGenerationWizard: () => ({
        type: types.CLOSE_CLUSTER_INIT_BUNDLE_GENERATION_WIZARD,
    }),
};

const clusterInitBundles = (state = [], action) => {
    if (action.type === types.FETCH_CLUSTER_INIT_BUNDLES.SUCCESS) {
        return isEqual(action.response.items, state) ? state : action.response.items;
    }
    return state;
};

const clusterInitBundleGenerationWizard = (state = null, { type, response }) => {
    switch (type) {
        case types.START_CLUSTER_INIT_BUNDLE_GENERATION_WIZARD:
            return { clusterInitBundle: '', helmValuesBundle: null, kubectlBundle: null };
        case types.CLOSE_CLUSTER_INIT_BUNDLE_GENERATION_WIZARD:
            return null;
        case types.GENERATE_CLUSTER_INIT_BUNDLE.SUCCESS:
            return {
                clusterInitBundle: response.meta,
                helmValuesBundle: response.helmValuesBundle,
                kubectlBundle: response.kubectlBundle,
            };
        default:
            return state;
    }
};

const reducer = combineReducers({
    clusterInitBundles,
    clusterInitBundleGenerationWizard,
});

const getClusterInitBundles = (state) => state.clusterInitBundles;
const clusterInitBundleGenerationWizardOpen = (state) => !!state.clusterInitBundleGenerationWizard;
const getCurrentGeneratedClusterInitBundle = (state) =>
    state.clusterInitBundleGenerationWizard
        ? state.clusterInitBundleGenerationWizard.clusterInitBundle
        : null;
const getCurrentGeneratedHelmValuesBundle = (state) =>
    state.clusterInitBundleGenerationWizard
        ? state.clusterInitBundleGenerationWizard.helmValuesBundle
        : null;
const getCurrentGeneratedKubectlBundle = (state) =>
    state.clusterInitBundleGenerationWizard
        ? state.clusterInitBundleGenerationWizard.kubectlBundle
        : null;

export const selectors = {
    getClusterInitBundles,
    clusterInitBundleGenerationWizardOpen,
    getCurrentGeneratedClusterInitBundle,
    getCurrentGeneratedHelmValuesBundle,
    getCurrentGeneratedKubectlBundle,
};

export default reducer;
