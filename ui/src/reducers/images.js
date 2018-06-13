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
    getSelectors as getSearchSelectors
} from 'reducers/pageSearch';

// Action types

export const types = {
    FETCH_IMAGES: createFetchingActionTypes('images/FETCH_IMAGES'),
    FETCH_IMAGE: createFetchingActionTypes('images/FETCH_IMAGE'),
    ...searchTypes('images')
};

// Actions

export const actions = {
    fetchImages: createFetchingActions(types.FETCH_IMAGES),
    fetchImage: createFetchingActions(types.FETCH_IMAGE),
    ...getSearchActions('images')
};

// Reducers

const bySha = (state = {}, action) => {
    if (action.response && action.response.entities && action.response.entities.image) {
        const imagesBySha = action.response.entities.image;
        const newState = mergeEntitiesById(state, imagesBySha);
        if (
            action.type === types.FETCH_IMAGES.SUCCESS &&
            (!action.params || !action.params.options || action.params.options.length === 0)
        ) {
            // fetched all images without any filter/search options, leave only those images
            const onlyExisting = pick(newState, Object.keys(imagesBySha));
            return isEqual(onlyExisting, state) ? state : onlyExisting;
        }
        return newState;
    }
    return state;
};

const filteredShas = (state = [], action) => {
    if (action.type === types.FETCH_IMAGES.SUCCESS) {
        return isEqual(action.response.result, state) ? state : action.response.result;
    }
    return state;
};

const reducer = combineReducers({
    bySha,
    filteredShas,
    ...searchReducers('images')
});

export default reducer;

// Selectors

const getImagesBySha = state => state.bySha;
const getFilteredShas = state => state.filteredShas;
const getFilteredImages = createSelector([getImagesBySha, getFilteredShas], (images, shas) =>
    shas.map(sha => images[sha])
);
const getImage = (state, sha) => getImagesBySha(state)[sha];

export const selectors = {
    getImagesBySha,
    getFilteredShas,
    getFilteredImages,
    getImage,
    ...getSearchSelectors('images')
};
