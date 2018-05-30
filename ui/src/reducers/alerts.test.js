import reducer, { actions } from './alerts';

const initialState = {
    byId: {},
    filteredIds: [],
    globalAlertCounts: [],
    alertCountsByPolicyCategories: [],
    alertCountsByCluster: [],
    alertsByTimeseries: [],
    searchModifiers: [],
    searchOptions: [],
    searchSuggestions: []
};

const alertsById = {
    alert1: { id: 'alert1' },
    alert2: { id: 'alert2' }
};
const alertsResponse = {
    entities: {
        alert: alertsById
    },
    result: {
        alerts: Object.keys(alertsById)
    }
};

const singleAlert = { id: 'alert1', data: 'data' };
const alertResponse = {
    entities: {
        alert: {
            [singleAlert.id]: singleAlert
        }
    }
};

describe('Alerts Reducer', () => {
    it('should return the initial state', () => {
        expect(reducer(undefined, {})).toEqual(initialState);
    });

    it('should add new alerts when received filtered alerts', () => {
        const prevState = {
            ...initialState,
            byId: {}
        };
        const nextState = reducer(prevState, actions.fetchAlerts.success(alertsResponse));
        expect(nextState.byId).toEqual({
            ...alertsById,
            ...prevState.byId
        });
        expect(nextState.filteredIds).toEqual(Object.keys(alertsById));
    });

    it('should enrich existing alert', () => {
        const prevState = {
            ...initialState,
            byId: alertsById
        };
        const nextState = reducer(prevState, actions.fetchAlert.success(alertResponse));

        expect(nextState.byId).toEqual({
            ...alertsById,
            [singleAlert.id]: singleAlert
        });
    });

    it('should cleanup non-existing violations when received new list of violations', () => {
        const prevState = {
            ...initialState,
            byId: {
                alert3: { id: 'alert3' }
            }
        };
        const nextState = reducer(prevState, actions.fetchAlerts.success(alertsResponse));
        expect(nextState.byId).toEqual(alertsById);
        expect(nextState.filteredIds).toEqual(Object.keys(alertsById));
    });
});
