import pageTypes from 'constants/pageTypes';
import useCases from 'constants/useCaseTypes';
import { generatePath, matchPath } from 'react-router-dom';
import qs from 'qs';
import { searchParams, sortParams, pagingParams } from 'constants/searchParams';
import { WorkflowState, WorkflowEntity } from './WorkflowStateManager';

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

function getTypeKeyFromParamValue(value, listOnly) {
    const listMatch = Object.entries(urlEntityListTypes).find(entry => entry[1] === value);
    const entityMatch = Object.entries(urlEntityTypes).find(entry => entry[1] === value);
    const match = listOnly ? listMatch : listMatch || entityMatch;
    return match ? match[0] : null;
}

// Convert workflowState and searchState to URL;
export function generateURL(workflowState) {
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

function getEntityFromURLParam(type, id) {
    return new WorkflowEntity(getTypeKeyFromParamValue(type), id);
}

export function paramsToStateStack(params) {
    const { pageEntityListType, pageEntityType, pageEntityId, entityId1, entityId2 } = params;
    const { entityType1: urlEntityType1, entityType2: urlEntityType2 } = params;
    const entityListType1 = getTypeKeyFromParamValue(urlEntityType1, true);
    const entityListType2 = getTypeKeyFromParamValue(urlEntityType2, true);
    const entityType1 = getTypeKeyFromParamValue(urlEntityType1);
    const entityType2 = getTypeKeyFromParamValue(urlEntityType2);
    const stateArray = [];
    if (!pageEntityListType && !pageEntityType) return stateArray;

    // List
    if (pageEntityListType) {
        stateArray.push(getEntityFromURLParam(pageEntityListType));

        if (entityId1) {
            stateArray.push(getEntityFromURLParam(pageEntityListType, entityId1));
        }
    } else {
        stateArray.push(getEntityFromURLParam(pageEntityType, pageEntityId));
        if (entityListType1) stateArray.push(new WorkflowEntity(entityListType1));
        if (entityType1 && entityId1) stateArray.push(new WorkflowEntity(entityType1, entityId1));
    }

    if (entityListType2) stateArray.push(new WorkflowEntity(entityListType2));
    if (entityType2 && entityId2) stateArray.push(new WorkflowEntity(entityType2, entityId2));

    return stateArray;
}

// Convert URL to workflow state and search objects
// note: this will read strictly from 'location' as 'match' is relative to the closest Route component
export function parseURL(location) {
    if (!location) return {};

    const { pathname, search } = location;
    const listParams = matchPath(pathname, {
        path: workflowPaths.LIST
    });
    const entityParams = matchPath(pathname, {
        path: workflowPaths.ENTITY
    });
    const dashboardParams = matchPath(pathname, {
        path: workflowPaths.DASHBOARD,
        exact: true
    });

    const { params } = entityParams || listParams || dashboardParams;
    const queryStr = search ? qs.parse(search, { ignoreQueryPrefix: true }) : {};

    const stateStackFromURLParams = paramsToStateStack(params) || [];

    let { workflowState: stateStackFromQueryString = [] } = queryStr;
    const {
        [searchParams.page]: pageSearch,
        [searchParams.sidePanel]: sidePanelSearch,
        [sortParams.page]: pageSort,
        [sortParams.sidePanel]: sidePanelSort,
        [pagingParams.page]: pagePaging,
        [pagingParams.sidePanel]: sidePanelPaging
    } = queryStr;

    stateStackFromQueryString = !Array.isArray(stateStackFromQueryString)
        ? [stateStackFromQueryString]
        : stateStackFromQueryString;
    stateStackFromQueryString = stateStackFromQueryString.map(
        ({ t, i }) => new WorkflowEntity(t, i)
    );

    const workflowState = new WorkflowState(
        params.context,
        [...stateStackFromURLParams, ...stateStackFromQueryString],
        {
            [searchParams.page]: pageSearch,
            [searchParams.sidePanel]: sidePanelSearch
        },
        {
            [sortParams.page]: pageSort,
            [sortParams.sidePanel]: sidePanelSort
        },
        {
            [pagingParams.page]: parseInt(pagePaging || 1, 10),
            [pagingParams.sidePanel]: parseInt(sidePanelPaging || 1, 10)
        }
    );

    return workflowState;
}

export function generateURLTo(workflowState, entityType, entityId) {
    if (!entityType && !entityId) return generateURL(workflowState);

    if (!entityId) {
        workflowState.pushList(entityType);
    } else if (!entityType) {
        workflowState.pushListItem(entityId);
    } else {
        workflowState.pushRelatedEntity(entityType, entityId);
    }
    return generateURL(workflowState);
}

export function generateURLToFromTable(workflowState, rowEntityId, entityType, entityId) {
    workflowState.pushListItem(rowEntityId);
    if (!entityId) {
        workflowState.pushList(entityType);
    } else {
        workflowState.pushRelatedEntity(entityType, entityId);
    }
    return generateURL(workflowState);
}
