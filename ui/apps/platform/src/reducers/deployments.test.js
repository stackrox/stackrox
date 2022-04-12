import reducer, { actions } from './deployments';

const initialState = {
    byId: {},
    filteredIds: [],
    searchModifiers: [],
    searchOptions: [],
    searchSuggestions: [],
};

const deploymentsById = {
    dep1: { id: 'dep1' },
    dep2: { id: 'dep2' },
};
const deploymentsResponse = {
    entities: {
        deployment: deploymentsById,
    },
    result: Object.keys(deploymentsById),
};

describe('Deployments Reducer', () => {
    it('should return the initial state', () => {
        expect(reducer(undefined, {})).toEqual(initialState);
    });

    it('should add new deployments when received filtered deployments', () => {
        const prevState = {
            ...initialState,
            byId: {
                dep3: { id: 'dep3' },
            },
        };
        const nextState = reducer(
            prevState,
            actions.fetchDeployments.success(deploymentsResponse, { options: ['opt'] })
        );
        expect(nextState.byId).toEqual({
            ...deploymentsById,
            ...prevState.byId,
        });
        expect(nextState.filteredIds).toEqual(Object.keys(deploymentsById));
    });

    it('should cleanup non-existing deployments when received new list of deployments', () => {
        const prevState = {
            ...initialState,
            byId: {
                dep3: { id: 'dep3' },
            },
        };
        const nextState = reducer(prevState, actions.fetchDeployments.success(deploymentsResponse));
        expect(nextState.byId).toEqual(deploymentsById);
        expect(nextState.filteredIds).toEqual(Object.keys(deploymentsById));
    });
});
