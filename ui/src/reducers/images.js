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

const byID = (state = {}, action) => {
    if (action.response && action.response.entities && action.response.entities.image) {
        const imagesByID = action.response.entities.image;
        const newState = mergeEntitiesById(state, imagesByID);
        if (
            action.type === types.FETCH_IMAGES.SUCCESS &&
            (!action.params || !action.params.options || action.params.options.length === 0)
        ) {
            // fetched all images without any filter/search options, leave only those images
            const onlyExisting = pick(newState, Object.keys(imagesByID));
            return isEqual(onlyExisting, state) ? state : onlyExisting;
        }
        return newState;
    }
    return state;
};

const filteredIDs = (state = [], action) => {
    if (action.type === types.FETCH_IMAGES.SUCCESS) {
        return isEqual(action.response.result, state) ? state : action.response.result;
    }
    return state;
};

const reducer = combineReducers({
    byID,
    filteredIDs,
    ...searchReducers('images')
});

export default reducer;

// Selectors

const getImagesByID = state => state.byID;
const getImages = createSelector(
    [getImagesByID],
    images => Object.values(images)
);
const getFilteredIDs = state => state.filteredIDs;
const getFilteredImages = createSelector(
    [getImagesByID, getFilteredIDs],
    (images, ids) => ids.map(id => images[id])
);
const getImage = (state, id) => getImagesByID(state)[id];

export const selectors = {
    getImages,
    getImagesByID,
    getFilteredIDs,
    getFilteredImages,
    getImage,
    ...getSearchSelectors('images')
};
