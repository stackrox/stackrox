import {
    routerReducer,
    LOCATION_CHANGE,
    push,
    replace,
    go,
    goBack,
    goForward,
} from 'react-router-redux';

// Action Types

export const types = {
    LOCATION_CHANGE,
};

// Actions

export const actions = {
    push,
    replace,
    go,
    goBack,
    goForward,
};

// Reducers

export default routerReducer;

// Selectors

const getLocation = (state) => state.location;

export const selectors = {
    getLocation,
};
