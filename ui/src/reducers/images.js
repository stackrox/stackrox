import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

// Action types

export const types = {
    FETCH_IMAGES: createFetchingActionTypes('images/FETCH_IMAGES')
};

// Actions

export const actions = {
    fetchImages: createFetchingActions(types.FETCH_IMAGES)
};

// Reducers

const images = (state = [], action) => {
    if (action.type === types.FETCH_IMAGES.SUCCESS) {
        return isEqual(action.response.images, state) ? state : action.response.images;
    }
    return state;
};

const reducer = combineReducers({
    images
});

export default reducer;

// Selectors

const getImages = state => state.images;

export const selectors = {
    getImages
};
