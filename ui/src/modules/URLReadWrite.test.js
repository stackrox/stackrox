import entityTypes from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';
import { urlEntityListTypes, urlEntityTypes } from '../routePaths';
import { parseURL, generateURL } from './URLReadWrite';
import { WorkflowEntity, WorkflowState } from './WorkflowStateManager';

function getLocation(search, pathname) {
    return {
        location: {
            search,
            pathname
        }
    };
}

const searchParams = {
    s: {
        sk1: 'v1',
        sk2: 'v2'
    },
    s2: {
        sk3: 'v3',
        sk4: 'v4'
    }
};

describe('ParseURL', () => {
    it('reads entity page workflow state params from url', () => {
        const search = {
            workflowState: [{ t: entityTypes.NAMESPACE }]
        };

        const { context, pageEntityType, pageEntityId } = {
            context: useCases.CONFIG_MANAGEMENT,
            pageEntityType: urlEntityTypes.CLUSTER,
            pageEntityId: '1234'
        };

        const pathname = `/main/${context}/${pageEntityType}/${pageEntityId}`;
        const { location } = getLocation(search, pathname);
        const workflowState = parseURL(location);

        // Test workflowState object
        expect(workflowState).not.toBeNull();
        expect(workflowState.useCase).toBe(useCases.CONFIG_MANAGEMENT);
        expect(workflowState.stateStack.length).toBe(search.workflowState.length + 1);
        expect(workflowState.stateStack[0]).toEqual({
            t: entityTypes.CLUSTER,
            i: pageEntityId
        });
        expect(workflowState.stateStack[1]).toEqual(search.workflowState[0]);
    });

    it('reads list page workflow state params from url', () => {
        const search = {
            workflowState: [{ t: entityTypes.CLUSTER, i: 'cluster1' }]
        };

        const { context, pageEntityListType } = {
            context: useCases.CONFIG_MANAGEMENT,
            pageEntityListType: urlEntityListTypes.CLUSTER
        };

        const pathname = `/main/${context}/${pageEntityListType}`;
        const { location } = getLocation(search, pathname);
        const workflowState = parseURL(location);

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

        const context = useCases.CONFIG_MANAGEMENT;
        const pathname = `/main/${context}/deployments`;

        const { location } = getLocation(search, pathname);

        const workflowState = parseURL(location);

        // Test workflowState object
        expect(workflowState.search).not.toBeNull();
        expect(workflowState.search).toEqual(searchParams);
        expect(workflowState.stateStack).toEqual([{ t: entityTypes.DEPLOYMENT }]);
    });
});

describe('GenerateURL', () => {
    it('generates a list url from workflowState', () => {
        const workflowState = new WorkflowState(
            useCases.COMPLIANCE,
            [
                new WorkflowEntity(entityTypes.NAMESPACE),
                new WorkflowEntity(entityTypes.NAMESPACE, 'nsId'),
                new WorkflowEntity(entityTypes.DEPLOYMENT)
            ],
            searchParams
        );

        const url = generateURL(workflowState, searchParams);
        expect(url).toBe(
            '/main/compliance/namespaces?workflowState[0][t]=NAMESPACE&workflowState[0][i]=nsId&workflowState[1][t]=DEPLOYMENT&s[sk1]=v1&s[sk2]=v2&s2[sk3]=v3&s2[sk4]=v4'
        );
    });

    it('generates a list url with sidepanel from workflowState', () => {
        const workflowState = new WorkflowState(
            useCases.COMPLIANCE,
            [
                new WorkflowEntity(entityTypes.NAMESPACE),
                new WorkflowEntity(entityTypes.NAMESPACE, 'nsId'),
                new WorkflowEntity(entityTypes.DEPLOYMENT),
                new WorkflowEntity(entityTypes.DEPLOYMENT, 'depId')
            ],
            searchParams
        );

        const url = generateURL(workflowState, searchParams);
        expect(url).toBe(
            '/main/compliance/namespaces?workflowState[0][t]=NAMESPACE&workflowState[0][i]=nsId&workflowState[1][t]=DEPLOYMENT&workflowState[2][t]=DEPLOYMENT&workflowState[2][i]=depId&s[sk1]=v1&s[sk2]=v2&s2[sk3]=v3&s2[sk4]=v4'
        );
    });

    it('generates an entity url from workflowState', () => {
        const workflowState = new WorkflowState(
            useCases.COMPLIANCE,
            [
                new WorkflowEntity(entityTypes.NAMESPACE, 'nsId'),
                new WorkflowEntity(entityTypes.DEPLOYMENT)
            ],
            searchParams
        );

        const url = generateURL(workflowState, searchParams);
        expect(url).toBe(
            '/main/compliance/namespace/nsId/deployments?s[sk1]=v1&s[sk2]=v2&s2[sk3]=v3&s2[sk4]=v4'
        );
    });

    it('generates an entity url with side panel from workflowState', () => {
        const workflowState = new WorkflowState(
            useCases.COMPLIANCE,
            [
                new WorkflowEntity(entityTypes.NAMESPACE, 'nsId'),
                new WorkflowEntity(entityTypes.DEPLOYMENT),
                new WorkflowEntity(entityTypes.DEPLOYMENT, 'depId')
            ],
            searchParams
        );

        const url = generateURL(workflowState, searchParams);
        expect(url).toBe(
            '/main/compliance/namespace/nsId/deployments?workflowState[0][t]=DEPLOYMENT&workflowState[0][i]=depId&s[sk1]=v1&s[sk2]=v2&s2[sk3]=v3&s2[sk4]=v4'
        );
    });

    it('generates a dashboard url from workflowState', () => {
        const workflowState = new WorkflowState(useCases.COMPLIANCE, [], searchParams);

        const url = generateURL(workflowState, searchParams);
        expect(url).toBe('/main/compliance?s[sk1]=v1&s[sk2]=v2&s2[sk3]=v3&s2[sk4]=v4');
    });
});
