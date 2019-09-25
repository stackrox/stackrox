import entityTypes from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';
import { urlEntityListTypes, urlEntityTypes } from '../routePaths';
import { parseURL, generateURL } from './URLReadWrite';

function getMatch(params) {
    return {
        params
    };
}

function getLocation(search) {
    return {
        search
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
    it('reads workflow state params from url', () => {
        const URLParams = {
            context: useCases.CONFIG_MANAGEMENT,
            pageEntityType: urlEntityTypes.CLUSTER,
            pageEntityId: 'pageEntityId',
            entityListType1: urlEntityListTypes.DEPLOYMENT,
            entityId1: 'entityId1',
            entityType2: urlEntityTypes.NAMESPACE,
            entityId2: 'entityId2'
        };

        const match = getMatch(URLParams);
        const location = getLocation(searchParams);

        const workflowStateParams = parseURL(match, location).params;
        expect(workflowStateParams.pageEntityType).toEqual(entityTypes.CLUSTER);
        expect(workflowStateParams.entityListType1).toEqual(entityTypes.DEPLOYMENT);
        expect(workflowStateParams.entityType2).toEqual(entityTypes.NAMESPACE);
    });

    // TODO: test translation of urlEntityListTypes to urlEntityTypes
});

describe('GenerateURL', () => {
    // TODO: use workflowStateManager to get a real workflow state objects
    it('generates an entity<>entity<>entity page url from workflowState', () => {
        const workflowState = {
            context: useCases.CONFIG_MANAGEMENT,
            pageEntityType: entityTypes.CLUSTER,
            pageEntityId: 'pageEntityId',
            entityType1: entityTypes.DEPLOYMENT,
            entityId1: 'entityId1',
            entityType2: entityTypes.NAMESPACE,
            entityId2: 'entityId2'
        };
        const url = generateURL(workflowState, searchParams);
        const queryString = `?s1[sk1]=${searchParams.s1.sk1}&s1[sk2]=${searchParams.s1.sk2}&sort1=${
            searchParams.sort1
        }&s2[sk3]=${searchParams.s2.sk3}&s2[sk4]=${searchParams.s2.sk4}&sort2=${
            searchParams.sort2
        }`;
        expect(url).toBe(
            `/main/configmanagement/cluster/pageEntityId/deployment/entityId1/namespace/entityId2${queryString}`
        );
    });

    it('generates an entity<>list<>entity page url from workflowState', () => {
        const workflowState = {
            context: useCases.CONFIG_MANAGEMENT,
            pageEntityType: entityTypes.CLUSTER,
            pageEntityId: 'pageEntityId',
            entityListType1: entityTypes.DEPLOYMENT,
            entityId1: 'entityId1',
            entityType2: entityTypes.NAMESPACE,
            entityId2: 'entityId2'
        };
        const url = generateURL(workflowState);
        expect(url).toBe(
            `/main/configmanagement/cluster/pageEntityId/deployments/entityId1/namespace/entityId2`
        );
    });

    it('generates an entity<>entity<>list page url from workflowState', () => {
        const workflowState = {
            context: useCases.CONFIG_MANAGEMENT,
            pageEntityType: entityTypes.CLUSTER,
            pageEntityId: 'pageEntityId',
            entityType1: entityTypes.DEPLOYMENT,
            entityId1: 'entityId1',
            entityListType2: entityTypes.NAMESPACE,
            entityId2: 'entityId2'
        };
        const url = generateURL(workflowState);
        expect(url).toBe(
            `/main/configmanagement/cluster/pageEntityId/deployment/entityId1/namespaces/entityId2`
        );
    });

    it('generates an entity<>list<>list page url from workflowState', () => {
        const workflowState = {
            context: useCases.CONFIG_MANAGEMENT,
            pageEntityType: entityTypes.CLUSTER,
            pageEntityId: 'pageEntityId',
            entityListType1: entityTypes.DEPLOYMENT,
            entityId1: 'entityId1',
            entityListType2: entityTypes.NAMESPACE,
            entityId2: 'entityId2'
        };
        const url = generateURL(workflowState);
        expect(url).toBe(
            `/main/configmanagement/cluster/pageEntityId/deployments/entityId1/namespaces/entityId2`
        );
    });

    it('generates an list<>entity page url from workflowState', () => {
        const workflowState = {
            context: useCases.CONFIG_MANAGEMENT,
            pageEntityListType: entityTypes.CLUSTER,
            entityId1: 'entityId1',
            entityType2: entityTypes.NAMESPACE,
            entityId2: 'entityId2'
        };
        const url = generateURL(workflowState);
        expect(url).toBe(`/main/configmanagement/clusters/entityId1/namespace/entityId2`);
    });

    it('generates an list<>list page url from workflowState', () => {
        const workflowState = {
            context: useCases.CONFIG_MANAGEMENT,
            pageEntityListType: entityTypes.CLUSTER,
            entityId1: 'entityId1',
            entityListType2: entityTypes.NAMESPACE,
            entityId2: 'entityId2'
        };
        const url = generateURL(workflowState);
        expect(url).toBe(`/main/configmanagement/clusters/entityId1/namespaces/entityId2`);
    });

    it('generates a query string from searchParams', () => {
        const workflowState = {
            context: useCases.CONFIG_MANAGEMENT,
            pageEntityType: entityTypes.CLUSTER,
            pageEntityId: 'pageEntityId',
            entityType1: entityTypes.DEPLOYMENT,
            entityId1: 'entityId1',
            entityType2: entityTypes.NAMESPACE,
            entityId2: 'entityId2'
        };
        const url = generateURL(workflowState, searchParams);
        const queryString = `?s1[sk1]=${searchParams.s1.sk1}&s1[sk2]=${searchParams.s1.sk2}&sort1=${
            searchParams.sort1
        }&s2[sk3]=${searchParams.s2.sk3}&s2[sk4]=${searchParams.s2.sk4}&sort2=${
            searchParams.sort2
        }`;
        expect(url).toEqual(expect.stringContaining(queryString));
    });
});
