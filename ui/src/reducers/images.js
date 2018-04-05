import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

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
    ...searchTypes('images')
};

// Actions

export const actions = {
    fetchImages: createFetchingActions(types.FETCH_IMAGES),
    ...getSearchActions('images')
};

// Reducers

const images = (state = [], action) => {
    if (action.type === types.FETCH_IMAGES.SUCCESS) {
        return isEqual(action.response.images, state) ? state : action.response.images;
    }
    return state;
};

const reducer = combineReducers({
    images,
    ...searchReducers('images')
});

export default reducer;

// Selectors

const getImages = state => state.images;

export const selectors = {
    getImages,
    ...getSearchSelectors('images')
};
