import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';
import pick from 'lodash/pick';
import { createSelector } from 'reselect';

import mergeEntitiesById from 'utils/mergeEntitiesById';
import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';
import {
    types as searchTypes,
    getActions as getSearchActions,
    reducers as searchReducers,
    getSelectors as getSearchSelectors,
} from 'reducers/pageSearch';

// Action types

export const types = {
    FETCH_DEPLOYMENTS: createFetchingActionTypes('deployments/FETCH_DEPLOYMENTS'),
    ...searchTypes('deployments'),
};

// Actions

export const actions = {
    fetchDeployments: createFetchingActions(types.FETCH_DEPLOYMENTS),
    ...getSearchActions('deployments'),
};

// Reducers

const byId = (state = {}, action) => {
    if (action.response && action.response.entities && action.response.entities.deployment) {
        const deploymentsById = action.response.entities.deployment;
        const newState = mergeEntitiesById(state, deploymentsById);
        if (
            action.type === types.FETCH_DEPLOYMENTS.SUCCESS &&
            (!action.params || !action.params.options || action.params.options.length === 0)
        ) {
            // fetched all deployments without any filter/search options, leave only those deployments
            const onlyExisting = pick(newState, Object.keys(deploymentsById));
            return isEqual(onlyExisting, state) ? state : onlyExisting;
        }
        return newState;
    }
    return state;
};

const filteredIds = (state = [], action) => {
    if (action.type === types.FETCH_DEPLOYMENTS.SUCCESS) {
        return isEqual(action.response.result, state) ? state : action.response.result;
    }
    return state;
};

const reducer = combineReducers({
    byId,
    filteredIds,
    ...searchReducers('deployments'),
});

export default reducer;

// Selectors

const getDeploymentsById = (state) => state.byId;
const getDeployments = createSelector([getDeploymentsById], (deployments) =>
    Object.values(deployments)
);
const getFilteredIds = (state) => state.filteredIds;
const getSelectedDeployment = (state, id) => getDeploymentsById(state)[id];
const getFilteredDeployments = createSelector(
    [getDeploymentsById, getFilteredIds],
    (deployments, ids) => ids.map((id) => deployments[id])
);

export const selectors = {
    getDeploymentsById,
    getDeployments,
    getSelectedDeployment,
    getFilteredDeployments,
    ...getSearchSelectors('deployments'),
};
