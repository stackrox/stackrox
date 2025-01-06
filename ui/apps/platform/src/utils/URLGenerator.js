import qs from 'qs';
import { generatePath } from 'react-router-dom';

import pageTypes from 'constants/pageTypes';
import { searchParams, sortParams, pagingParams } from 'constants/searchParams';
import useCases from 'constants/useCaseTypes';
import {
    workflowPaths,
    clustersBasePath,
    clustersPathWithParam,
    riskPath,
    secretsPath,
    urlEntityListTypes,
    urlEntityTypes,
    policiesPath,
} from '../routePaths';

const defaultPathMap = {
    [pageTypes.DASHBOARD]: workflowPaths.DASHBOARD,
    [pageTypes.ENTITY]: workflowPaths.ENTITY,
    [pageTypes.LIST]: workflowPaths.LIST,
};

const legacyPathMap = {
    [useCases.CLUSTERS]: {
        [pageTypes.ENTITY]: clustersPathWithParam,
        [pageTypes.LIST]: clustersBasePath,
        [pageTypes.DASHBOARD]: clustersBasePath,
    },
    [useCases.RISK]: {
        [pageTypes.ENTITY]: riskPath,
        [pageTypes.LIST]: '/main/risk',
        [pageTypes.DASHBOARD]: '/main/risk',
    },
    [useCases.SECRET]: {
        [pageTypes.ENTITY]: secretsPath,
        [pageTypes.LIST]: '/main/configmanagement/secrets',
        [pageTypes.DASHBOARD]: '/main/configmanagement/secrets',
    },
    [useCases.POLICY]: {
        [pageTypes.ENTITY]: policiesPath,
        [pageTypes.LIST]: '/main/policies',
        [pageTypes.DASHBOARD]: '/main/policies',
    },
};
function generateURL(workflowState) {
    const { stateStack: originalStateStack, useCase } = workflowState;
    const stateStack = [...originalStateStack];
    const pageStack = workflowState.getPageStack();
    const qsStack = stateStack.slice(pageStack.length);
    if (!useCase) {
        throw new Error('Cannot generate a url from workflowState without a use case');
    }

    // Find the path map for the use case
    const pathMap = legacyPathMap[useCase] || defaultPathMap;
    if (!pathMap) {
        throw new Error(`Can't generate a URL. No paths found for context ${useCase}`);
    }

    const pageParams = workflowState.getPageStack();

    // determine the page type
    let pageType = pageTypes.DASHBOARD;
    if (pageParams.length > 0) {
        pageType = pageParams[0].entityId ? pageTypes.ENTITY : pageTypes.LIST;
    }

    // determine the path
    const path = pathMap[pageType];
    if (!path) {
        throw new Error(
            `Can't generate a URL. No path found for context ${useCase} and page type ${pageType}`
        );
    }

    // create url params
    const params = { useCase, context: useCase }; // using legacy context url param. remove after paths are updated
    if (pageParams.length > 0) {
        params.pageEntityId = pageParams[0].entityId;
        params.pageEntityType = urlEntityTypes[pageParams[0].entityType];
        params.pageEntityListType = urlEntityListTypes[pageParams[0].entityType];
        if (pageType === pageTypes.ENTITY && pageParams[1]) {
            params.entityType1 = urlEntityListTypes[pageParams[1].entityType];
        }
    }

    // Add url params for legacy contexts
    if (useCase === useCases.CLUSTERS) {
        params.clusterId = params.pageEntityId;
    } else if (useCase === useCases.SECRET) {
        params.secretId = params.pageEntityId;
    } else if (useCase === useCases.RISK) {
        params.deploymentId = params.pageEntityId;
    }

    // generate the querystring using remaining statestack params
    const queryParams = {
        workflowState: qsStack,
        [searchParams.page]: workflowState.search[searchParams.page],
        [searchParams.sidePanel]: workflowState.search[searchParams.sidePanel],
        [sortParams.page]: workflowState.sort[sortParams.page],
        [sortParams.sidePanel]: workflowState.sort[sortParams.sidePanel],
        [pagingParams.page]: workflowState.paging[pagingParams.page],
        [pagingParams.sidePanel]: workflowState.paging[pagingParams.sidePanel],
    };

    // Don't write URLs with p=0 or p2=0, since that's the default value anyway
    if (queryParams[pagingParams.page] === 0) {
        delete queryParams[pagingParams.page];
    }
    if (queryParams[pagingParams.sidePanel] === 0) {
        delete queryParams[pagingParams.sidePanel];
    }

    // Don't write URLs with s1 or s2 empty, since that's superfluous
    if (!queryParams[searchParams.page]) {
        delete queryParams[searchParams.page];
    }
    if (!queryParams[searchParams.sidePanel]) {
        delete queryParams[searchParams.sidePanel];
    }

    // Don't write URLs with sort or sort2 empty, since that's superfluous
    if (!queryParams[sortParams.page]) {
        delete queryParams[sortParams.page];
    }
    if (!queryParams[sortParams.sidePanel]) {
        delete queryParams[sortParams.sidePanel];
    }

    // hybrid approach to using page params in Config Mgmt, but keeping entities in URL params
    if (useCase === useCases.CONFIG_MANAGEMENT) {
        const stateToDowngrade = queryParams?.workflowState;
        if (stateToDowngrade) {
            const entityId1 = stateToDowngrade[0] && stateToDowngrade[0].i;
            const entityType2 = stateToDowngrade[1] && stateToDowngrade[1].t;
            const entityId2 = stateToDowngrade[1] && stateToDowngrade[1].i;

            params.entityId1 = entityId1;
            params.entityType2 = urlEntityListTypes[entityType2];
            params.entityId2 = entityId2;

            // @ts-ignore The operand of a 'delete' operator must be optional.ts (2790)
            delete queryParams.workflowState;
        }
    }

    const queryString = queryParams
        ? qs.stringify(queryParams, {
              addQueryPrefix: true,
              arrayFormat: 'indices',
              encodeValuesOnly: true,
          })
        : '';

    const encodedParams = Object.fromEntries(
        Object.entries(params).map(([key, value]) => [key, encodeURIComponent(value)])
    );
    const newPath = generatePath(path, encodedParams) + queryString;
    return newPath;
}

export default generateURL;
