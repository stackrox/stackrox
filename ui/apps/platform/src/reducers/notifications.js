import { combineReducers } from 'redux';

// Action types

export const types = {
    ADD_NOTIFICATION: 'notifications/ADD_NOTIFICATION',
    REMOVE_OLDEST_NOTIFICATION: 'notifications/REMOVE_OLDEST_NOTIFICATION',
};

// Actions

export const actions = {
    addNotification: (notification) => ({ type: types.ADD_NOTIFICATION, notification }),
    removeOldestNotification: () => ({ type: types.REMOVE_OLDEST_NOTIFICATION }),
};

// Reducers

const notifications = (state = [], action) => {
    const newState = state.slice();
    switch (action.type) {
        case types.ADD_NOTIFICATION:
            newState.push(action.notification);
            return newState;
        case types.REMOVE_OLDEST_NOTIFICATION:
            newState.shift();
            return newState;
        default:
            return state;
    }
};

const reducer = combineReducers({
    notifications,
});

export default reducer;

// Selectors

const getNotifications = (state) => state.notifications;

export const selectors = {
    getNotifications,
};
