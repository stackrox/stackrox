import qs from 'qs';

import entityTypes from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';
import { searchParams, sortParams, pagingParams } from 'constants/searchParams';
import { urlEntityListTypes, urlEntityTypes } from '../routePaths';
import parseURL from './URLParser';
import generateURL from './URLGenerator';

import WorkflowEntity from './WorkflowEntity';
import { WorkflowState } from './WorkflowState';

function getLocation(search, pathname) {
    return {
        location: {
            pathname,
            search: `?${qs.stringify(search)}`,
        },
    };
}

const searchParamValues = {
    [searchParams.page]: {
        sk1: 'v1',
        sk2: 'v2',
    },
    [searchParams.sidePanel]: {
        sk3: 'v3',
        sk4: 'v4',
    },
};

const emptySearchParamValues = {
    [searchParams.page]: '',
    [searchParams.sidePanel]: '',
};

const sortParamValues = {
    [sortParams.page]: [{ id: 'name1', desc: true }],
    [sortParams.sidePanel]: [{ id: 'name2', desc: false }],
};

const pagingParamValues = {
    [pagingParams.page]: 1,
    [pagingParams.sidePanel]: 2,
};

describe('ParseURL', () => {
    it('reads entity page workflow state params from url', () => {
        const search = {
            workflowState: [{ t: entityTypes.NAMESPACE }],
        };

        const { context, pageEntityType, pageEntityId } = {
            context: useCases.CONFIG_MANAGEMENT,
            pageEntityType: urlEntityTypes.CLUSTER,
            pageEntityId: '1234',
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
            i: pageEntityId,
        });
        expect(workflowState.stateStack[1]).toEqual(search.workflowState[0]);
    });

    it('reads list page workflow state params from url', () => {
        const search = {
            workflowState: [{ t: entityTypes.CLUSTER, i: 'cluster1' }],
        };

        const { context, pageEntityListType } = {
            context: useCases.CONFIG_MANAGEMENT,
            pageEntityListType: urlEntityListTypes.CLUSTER,
        };

        const pathname = `/main/${context}/${pageEntityListType}`;
        const { location } = getLocation(search, pathname);
        const workflowState = parseURL(location);

        // Test workflowState object
        expect(workflowState).not.toBeNull();
        expect(workflowState.useCase).toBe(useCases.CONFIG_MANAGEMENT);
        expect(workflowState.stateStack.length).toBe(search.workflowState.length + 1);
        expect(workflowState.stateStack[0]).toEqual({
            t: entityTypes.CLUSTER,
        });
        expect(workflowState.stateStack[1]).toEqual(search.workflowState[0]);
    });

    it('reads query params from url', () => {
        const search = {
            ...searchParamValues,
            ...sortParamValues,
            ...pagingParamValues,
        };

        const context = useCases.CONFIG_MANAGEMENT;
        const pathname = `/main/${context}/deployments`;

        const { location } = getLocation(search, pathname);

        const workflowState = parseURL(location);

        // Test workflowState object
        expect(workflowState.search).toEqual(searchParamValues);
        expect(workflowState.sort).toEqual(sortParamValues);
        expect(workflowState.paging).toEqual(pagingParamValues);

        expect(workflowState.stateStack).toEqual([{ t: entityTypes.DEPLOYMENT }]);
    });

    it('reads entity page non-workflow state params from url', () => {
        const context = 'clusters';
        const pathname = `/main/${context}/0123456789abcdef`;

        const location = {
            pathname,
            search: '',
        };

        const workflowState = parseURL(location);

        // Test workflowState object
        expect(workflowState).not.toBeNull();
        expect(workflowState.useCase).toBe(context);
        expect(workflowState.stateStack.length).toBe(0);
        expect(workflowState.search).toEqual({ s: null, s2: null });
        expect(workflowState.sort).toEqual({ sort: null, sort2: null });
        expect(workflowState.paging).toEqual({ p: 0, p2: 0 });
    });

    it('reads list page non-workflow state including query params from url', () => {
        const context = 'clusters';
        const pathname = `/main/${context}`;

        const location = {
            pathname,
            search: 's[CLUSTER_HEALTH]=HEALTHY',
        };

        const workflowState = parseURL(location);

        // Test workflowState object
        expect(workflowState).not.toBeNull();
        expect(workflowState.useCase).toBe(context);
        expect(workflowState.stateStack.length).toBe(0);
        expect(workflowState.search).toEqual({ s: { CLUSTER_HEALTH: 'HEALTHY' }, s2: null });
        expect(workflowState.sort).toEqual({ sort: null, sort2: null });
        expect(workflowState.paging).toEqual({ p: 0, p2: 0 });
    });
});

describe('GenerateURL', () => {
    it('generates a url with no search params when the search param objects are empty', () => {
        const workflowState = new WorkflowState(
            useCases.COMPLIANCE,
            [
                new WorkflowEntity(entityTypes.NAMESPACE),
                new WorkflowEntity(entityTypes.NAMESPACE, 'nsId'),
                new WorkflowEntity(entityTypes.DEPLOYMENT),
            ],
            emptySearchParamValues,
            sortParamValues,
            pagingParamValues
        );

        const url = generateURL(workflowState);
        expect(url).toBe(
            '/main/compliance/namespaces?workflowState[0][t]=NAMESPACE&workflowState[0][i]=nsId&workflowState[1][t]=DEPLOYMENT&sort[0][id]=name1&sort[0][desc]=true&sort2[0][id]=name2&sort2[0][desc]=false&p=1&p2=2'
        );
    });
    it('generates a list url from workflowState', () => {
        const workflowState = new WorkflowState(
            useCases.COMPLIANCE,
            [
                new WorkflowEntity(entityTypes.NAMESPACE),
                new WorkflowEntity(entityTypes.NAMESPACE, 'nsId'),
                new WorkflowEntity(entityTypes.DEPLOYMENT),
            ],
            searchParamValues,
            sortParamValues,
            pagingParamValues
        );

        const url = generateURL(workflowState);
        expect(url).toBe(
            '/main/compliance/namespaces?workflowState[0][t]=NAMESPACE&workflowState[0][i]=nsId&workflowState[1][t]=DEPLOYMENT&s[sk1]=v1&s[sk2]=v2&s2[sk3]=v3&s2[sk4]=v4&sort[0][id]=name1&sort[0][desc]=true&sort2[0][id]=name2&sort2[0][desc]=false&p=1&p2=2'
        );
    });

    it('generates a list url with sidepanel from workflowState', () => {
        const workflowState = new WorkflowState(
            useCases.COMPLIANCE,
            [
                new WorkflowEntity(entityTypes.NAMESPACE),
                new WorkflowEntity(entityTypes.NAMESPACE, 'nsId'),
                new WorkflowEntity(entityTypes.DEPLOYMENT),
                new WorkflowEntity(entityTypes.DEPLOYMENT, 'depId'),
            ],
            searchParamValues,
            sortParamValues,
            pagingParamValues
        );

        const url = generateURL(workflowState);
        expect(url).toBe(
            '/main/compliance/namespaces?workflowState[0][t]=NAMESPACE&workflowState[0][i]=nsId&workflowState[1][t]=DEPLOYMENT&workflowState[2][t]=DEPLOYMENT&workflowState[2][i]=depId&s[sk1]=v1&s[sk2]=v2&s2[sk3]=v3&s2[sk4]=v4&sort[0][id]=name1&sort[0][desc]=true&sort2[0][id]=name2&sort2[0][desc]=false&p=1&p2=2'
        );
    });

    it('generates an entity url from workflowState', () => {
        const workflowState = new WorkflowState(
            useCases.COMPLIANCE,
            [
                new WorkflowEntity(entityTypes.NAMESPACE, 'nsId'),
                new WorkflowEntity(entityTypes.DEPLOYMENT),
            ],
            searchParamValues,
            sortParamValues,
            pagingParamValues
        );

        const url = generateURL(workflowState);
        expect(url).toBe(
            '/main/compliance/namespace/nsId/deployments?s[sk1]=v1&s[sk2]=v2&s2[sk3]=v3&s2[sk4]=v4&sort[0][id]=name1&sort[0][desc]=true&sort2[0][id]=name2&sort2[0][desc]=false&p=1&p2=2'
        );
    });

    it('generates an entity url with side panel from workflowState', () => {
        const workflowState = new WorkflowState(
            useCases.COMPLIANCE,
            [
                new WorkflowEntity(entityTypes.NAMESPACE, 'nsId'),
                new WorkflowEntity(entityTypes.DEPLOYMENT),
                new WorkflowEntity(entityTypes.DEPLOYMENT, 'depId'),
            ],
            searchParamValues,
            sortParamValues,
            pagingParamValues
        );

        const url = generateURL(workflowState);
        expect(url).toBe(
            '/main/compliance/namespace/nsId/deployments?workflowState[0][t]=DEPLOYMENT&workflowState[0][i]=depId&s[sk1]=v1&s[sk2]=v2&s2[sk3]=v3&s2[sk4]=v4&sort[0][id]=name1&sort[0][desc]=true&sort2[0][id]=name2&sort2[0][desc]=false&p=1&p2=2'
        );
    });

    it('generates a dashboard url from workflowState', () => {
        const workflowState = new WorkflowState(useCases.COMPLIANCE, []);

        const url = generateURL(workflowState);
        expect(url).toBe('/main/compliance');
    });
});
