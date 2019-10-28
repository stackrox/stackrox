import qs from 'qs';
import pageTypes from 'constants/pageTypes';
import useCases from 'constants/useCaseTypes';
import { searchParams, sortParams, pagingParams } from 'constants/searchParams';
import { generatePath } from 'react-router-dom';
import {
    nestedPaths as workflowPaths,
    riskPath,
    secretsPath,
    urlEntityListTypes,
    urlEntityTypes,
    policiesPath
} from '../routePaths';

const defaultPathMap = {
    [pageTypes.DASHBOARD]: workflowPaths.DASHBOARD,
    [pageTypes.ENTITY]: workflowPaths.ENTITY,
    [pageTypes.LIST]: workflowPaths.LIST
};

const legacyPathMap = {
    [useCases.RISK]: {
        [pageTypes.ENTITY]: riskPath,
        [pageTypes.LIST]: '/main/risk',
        [pageTypes.DASHBOARD]: '/main/risk'
    },
    [useCases.SECRET]: {
        [pageTypes.ENTITY]: secretsPath,
        [pageTypes.LIST]: '/main/configmanagement/secrets',
        [pageTypes.DASHBOARD]: '/main/configmanagement/secrets'
    },
    [useCases.POLICY]: {
        [pageTypes.ENTITY]: policiesPath,
        [pageTypes.LIST]: '/main/policies',
        [pageTypes.DASHBOARD]: '/main/policies'
    }
};
function generateURL(workflowState) {
    const { stateStack: originalStateStack, useCase } = workflowState;
    const stateStack = [...originalStateStack];
    const pageStack = workflowState.getPageStack();
    const qsStack = stateStack.slice(pageStack.length);
    if (!useCase) throw new Error('Cannot generate a url from workflowState without a use case');

    // Find the path map for the use case
    const pathMap = legacyPathMap[useCase] || defaultPathMap;
    if (!pathMap) throw new Error(`Can't generate a URL. No paths found for context ${useCase}`);

    const pageParams = workflowState.getPageStack();

    // determine the page type
    let pageType = pageTypes.DASHBOARD;
    if (pageParams.length > 0)
        pageType = pageParams[0].entityId ? pageTypes.ENTITY : pageTypes.LIST;

    // determine the path
    const path = pathMap[pageType];
    if (!path)
        throw new Error(
            `Can't generate a URL. No path found for context ${useCase} and page type ${pageType}`
        );

    // create url params
    const params = { useCase, context: useCase }; // using legacy context url param. remove after paths are updated
    if (pageParams.length > 0) {
        params.pageEntityId = pageParams[0].entityId;
        params.pageEntityType = urlEntityTypes[pageParams[0].entityType];
        params.pageEntityListType = urlEntityListTypes[pageParams[0].entityType];
        if (pageType === pageTypes.ENTITY && pageParams[1])
            params.entityType1 = urlEntityListTypes[pageParams[1].entityType];
    }

    // Add url params for legacy contexts
    if (useCase === useCases.SECRET) {
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
        [pagingParams.sidePanel]: workflowState.paging[pagingParams.sidePanel]
    };

    const queryString = queryParams
        ? qs.stringify(queryParams, {
              addQueryPrefix: true,
              arrayFormat: 'indices',
              encodeValuesOnly: true
          })
        : '';

    return generatePath(path, params) + queryString;
}

export default generateURL;
