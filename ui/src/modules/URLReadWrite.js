import pageTypes from 'constants/pageTypes';
import useCases from 'constants/useCaseTypes';
import { generatePath, matchPath } from 'react-router-dom';
import qs from 'qs';
import { WorkflowState, isStackValid, WorkflowEntity } from './WorkflowStateManager';

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
export function generateURL(workflowState, searchState) {
    const { stateStack: originalStateStack, useCase } = workflowState;
    const stateStack = [...originalStateStack];

    if (!useCase) throw new Error('Cannot generate a url from workflowState without a use case');

    // Find the path map for the use case
    const pathMap = legacyPathMap[useCase] || defaultPathMap;
    if (!pathMap) throw new Error(`Can't generate a URL. No paths found for context ${useCase}`);

    const pageParams = stateStack.shift();

    // determine the page type
    let pageType = pageTypes.DASHBOARD;
    if (pageParams) pageType = pageParams.i ? pageTypes.ENTITY : pageTypes.LIST;

    // determine the path
    const path = pathMap[pageType];
    if (!path)
        throw new Error(
            `Can't generate a URL. No path found for context ${useCase} and page type ${pageType}`
        );

    // create url params
    const params = { useCase, context: useCase }; // using legacy context url param. remove after paths are updated
    if (pageParams) {
        params.pageEntityId = pageParams.i;
        params.pageEntityType = urlEntityTypes[pageParams.t];
        params.pageEntityListType = urlEntityListTypes[pageParams.t];
    }

    // Add url params for legacy contexts
    if (useCase === useCases.SECRET) {
        params.secretId = params.pageEntityId;
    } else if (useCase === useCases.RISK) {
        params.deploymentId = params.pageEntityId;
    }

    // generate the querystring using remaining statestack params
    const queryParams = { workflowState: stateStack, ...searchState };

    const queryString = queryParams
        ? qs.stringify(queryParams, {
              addQueryPrefix: true,
              arrayFormat: 'indices',
              encodeValuesOnly: true
          })
        : '';

    return generatePath(path, params) + queryString;
}

function getStateArrayObject(type, entityId) {
    if (!type && !entityId) return null;
    const obj = new WorkflowEntity(type);
    if (entityId) obj.i = entityId;

    return obj;
}

export function paramsToStateStack(params) {
    const {
        pageEntityListType,
        pageEntityType,
        pageEntityId,
        entityId1,
        entityId2,
        entityType1,
        entityType2,
        entityListType1,
        entityListType2
    } = params;

    const stateArray = [];
    if (!pageEntityListType && !pageEntityType) return stateArray;

    if (pageEntityListType)
        stateArray.push(new WorkflowEntity(getTypeKeyFromParamValue(pageEntityListType)));
    else
        stateArray.push(new WorkflowEntity(getTypeKeyFromParamValue(pageEntityType), pageEntityId));

    const tab = entityListType1
        ? new WorkflowEntity(getTypeKeyFromParamValue(entityListType1))
        : null;
    const entityTypeKey1 =
        entityId1 && getTypeKeyFromParamValue(entityType1 || entityListType1 || pageEntityListType);
    const entity1 = getStateArrayObject(entityTypeKey1, entityId1);

    const list = entityListType2
        ? new WorkflowEntity(getTypeKeyFromParamValue(entityListType2))
        : null;
    const entityTypeKey2 = getTypeKeyFromParamValue(entityType2 || entityListType2);
    const entity2 = getStateArrayObject(entityTypeKey2, entityId2);
    // TODO: make this work
    if (tab) stateArray.push(tab);
    if (entity1) stateArray.push(entity1);
    if (list) stateArray.push(list);
    if (entity2) stateArray.push(entity2);

    if (!isStackValid)
        throw new Error('The supplied workflow state params produce an invalid state');

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

    let stateStack = paramsToStateStack(params) || [];
    const query = search ? qs.parse(search, { ignoreQueryPrefix: true }) : {};
    const { workflowState: urlWorkflowState = [], ...searchState } = query;

    const arrayState = !Array.isArray(urlWorkflowState) ? [urlWorkflowState] : urlWorkflowState;
    const urlWorkflowStateStack = arrayState.map(({ t, i }) => new WorkflowEntity(t, i));

    // if on dashboard, the workflowState query params should be ignored
    stateStack = dashboardParams ? [] : [...stateStack, ...urlWorkflowStateStack];
    const workflowState = new WorkflowState(params.context, stateStack);

    // Convert URL parameter values to enum types
    // if (params.pageEntityListType) {
    //     stateStack.unshift({ t: getTypeKeyFromParamValue(params.pageEntityListType) });
    // } else if (params.pageEntityType) {
    //     stateStack.unshift({
    //         t: getTypeKeyFromParamValue(params.pageEntityType),
    //         i: params.pageEntityId
    //     });
    // }

    return {
        workflowState,
        searchState
    };
}
