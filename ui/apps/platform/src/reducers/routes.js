import { LOCATION_CHANGE, push, replace, go, goBack, goForward } from 'connected-react-router';

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

// Selectors

const getLocation = (state) => state.location;

export const selectors = {
    getLocation,
};
