import { Reducer, combineReducers } from 'redux';

// Action types

export type AddNotificationAction = {
    type: 'notifications/ADD_NOTIFICATION';
    notification: string;
};

export type RemoveOldestNotificationAction = {
    type: 'notifications/REMOVE_OLDEST_NOTIFICATION';
};

export type NotificationAction = AddNotificationAction | RemoveOldestNotificationAction;

// Action creator functions

export const actions = {
    addNotification: (notification: string): AddNotificationAction => ({
        type: 'notifications/ADD_NOTIFICATION',
        notification,
    }),
    removeOldestNotification: (): RemoveOldestNotificationAction => ({
        type: 'notifications/REMOVE_OLDEST_NOTIFICATION',
    }),
};

// Reducers

const notifications: Reducer<string[], NotificationAction> = (state = [], action) => {
    const newState = state.slice();
    switch (action.type) {
        case 'notifications/ADD_NOTIFICATION':
            newState.push(action.notification);
            return newState;
        case 'notifications/REMOVE_OLDEST_NOTIFICATION':
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

type State = ReturnType<typeof reducer>;

const notificationsSelector = (state: State) => state.notifications;

export const selectors = {
    notificationsSelector,
};
