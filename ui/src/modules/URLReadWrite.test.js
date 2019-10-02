import entityTypes from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';
import { urlEntityListTypes, urlEntityTypes } from '../routePaths';
import { parseURL, generateURL } from './URLReadWrite';

function getMatchLocation(params, search) {
    return {
        match: {
            params
        },
        location: {
            search
        }
    };
}

const searchParams = {
    s1: {
        sk1: 'v1',
        sk2: 'v2'
    },
    sort1: 's1',
    s2: {
        sk3: 'v3',
        sk4: 'v4'
    },
    sort2: 's2'
};

describe('ParseURL', () => {
    it('reads entity page workflow state params from url', () => {
        const search = {
            workflowState: [{ t: entityTypes.NAMESPACE }]
        };

        const params = {
            context: useCases.CONFIG_MANAGEMENT,
            pageEntityType: urlEntityTypes.CLUSTER,
            pageEntityId: 'pageEntityId'
        };

        const { match, location } = getMatchLocation(params, search);

        const { workflowState } = parseURL(match, location);

        // Test workflowState object
        expect(workflowState).not.toBeNull();
        expect(workflowState.useCase).toBe(useCases.CONFIG_MANAGEMENT);
        expect(workflowState.stateStack.length).toBe(search.workflowState.length + 1);
        expect(workflowState.stateStack[0]).toEqual({
            t: entityTypes.CLUSTER,
            i: params.pageEntityId
        });
        expect(workflowState.stateStack[1]).toEqual(search.workflowState[0]);
    });

    it('reads list page workflow state params from url', () => {
        const search = {
            workflowState: [{ t: entityTypes.CLUSTER, i: 'cluster1' }]
        };

        const params = {
            context: useCases.CONFIG_MANAGEMENT,
            pageEntityListType: urlEntityListTypes.CLUSTER
        };

        const { match, location } = getMatchLocation(params, search);

        const { workflowState } = parseURL(match, location);

        // Test workflowState object
        expect(workflowState).not.toBeNull();
        expect(workflowState.useCase).toBe(useCases.CONFIG_MANAGEMENT);
        expect(workflowState.stateStack.length).toBe(search.workflowState.length + 1);
        expect(workflowState.stateStack[0]).toEqual({
            t: entityTypes.CLUSTER
        });
        expect(workflowState.stateStack[1]).toEqual(search.workflowState[0]);
    });

    it('reads query params from url', () => {
        const search = {
            ...searchParams
        };

        const params = {
            context: useCases.CONFIG_MANAGEMENT
        };

        const { match, location } = getMatchLocation(params, search);

        const { searchState } = parseURL(match, location);

        // Test workflowState object
        expect(searchState).not.toBeNull();
        expect(searchState).toEqual(searchParams);
    });
});

describe('GenerateURL', () => {
    it('generates a list url from workflowState', () => {
        const workflowState = {
            useCase: useCases.COMPLIANCE,
            stateStack: [
                { t: entityTypes.NAMESPACE },
                { t: entityTypes.NAMESPACE, i: 'nsId' },
                { t: entityTypes.DEPLOYMENT }
            ]
        };

        const url = generateURL(workflowState, searchParams);
        expect(url).toBe(
            '/main/compliance/namespaces?workflowState[t]=NAMESPACE&workflowState[i]=nsId&workflowState[t]=DEPLOYMENT&s1[sk1]=v1&s1[sk2]=v2&sort1=s1&s2[sk3]=v3&s2[sk4]=v4&sort2=s2'
        );
    });

    it('generates an entity url from workflowState', () => {
        const workflowState = {
            useCase: useCases.COMPLIANCE,
            stateStack: [{ t: entityTypes.NAMESPACE, i: 'nsId' }, { t: entityTypes.DEPLOYMENT }]
        };

        const url = generateURL(workflowState, searchParams);
        expect(url).toBe(
            '/main/compliance/namespace/nsId?workflowState[t]=DEPLOYMENT&s1[sk1]=v1&s1[sk2]=v2&sort1=s1&s2[sk3]=v3&s2[sk4]=v4&sort2=s2'
        );
    });

    it('generates a dashboard url from workflowState', () => {
        const workflowState = {
            useCase: useCases.COMPLIANCE,
            stateStack: []
        };

        const url = generateURL(workflowState, searchParams);
        expect(url).toBe(
            '/main/compliance?s1[sk1]=v1&s1[sk2]=v2&sort1=s1&s2[sk3]=v3&s2[sk4]=v4&sort2=s2'
        );
    });
});
