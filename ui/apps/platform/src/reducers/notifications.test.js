import reducer, { actions } from './notifications';

const initialState = {
    notifications: [],
};

const notification = 'notification added';

describe('Notifications Reducer', () => {
    it('should return the initial state', () => {
        expect(reducer(undefined, {})).toEqual(initialState);
    });

    it('should add new notifications when receiving a new notification', () => {
        const prevState = initialState;
        const nextState = reducer(prevState, actions.addNotification(notification));
        expect(nextState.notifications).toEqual([notification]);
    });

    it('should remove oldest notification', () => {
        const prevState = {
            notifications: [notification],
        };
        const nextState = reducer(prevState, actions.removeOldestNotification());
        expect(nextState.notifications).toEqual([]);
    });
});
