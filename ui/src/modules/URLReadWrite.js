import pageTypes from 'constants/pageTypes';
import useCases from 'constants/useCaseTypes';
import { generatePath } from 'react-router-dom';
import qs from 'qs';
import {
    nestedPaths as workflowPaths,
    riskPath,
    secretsPath,
    urlEntityListTypes,
    urlEntityTypes,
    policiesPath
} from '../routePaths';

function getTypeKeyFromParamValue(value, listOnly) {
    const listMatch = Object.entries(urlEntityListTypes).find(entry => entry[1] === value);
    const entityMatch = Object.entries(urlEntityTypes).find(entry => entry[1] === value);
    const match = listOnly ? listMatch : listMatch || entityMatch;
    return match ? match[0] : null;
}

function isListType(value) {
    return Object.values(urlEntityListTypes).includes(value);
}

export function getPageType(workflowState) {
    if (workflowState.pageEntityListType) {
        return pageTypes.LIST;
    }
    if (workflowState.pageEntityType) {
        return pageTypes.ENTITY;
    }
    return pageTypes.DASHBOARD;
}

export function generateURL(workflowState, queryParams) {
    const pageType = getPageType(workflowState);
    const { context } = workflowState;

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

    const contextPaths = legacyPathMap[context] || defaultPathMap;
    if (!contextPaths)
        throw new Error(`Can't generate a URL. No paths found for context ${context}`);

    const path = contextPaths[pageType];
    if (!path)
        throw new Error(
            `Can't generate a URL. No path found for context ${context} and page type ${pageType}`
        );

    const params = { ...workflowState };

    // Patching url params for legacy contexts
    if (context === useCases.SECRET) {
        params.secretId = params.pageEntityId;
    } else if (context === useCases.RISK) {
        params.deploymentId = params.pageEntityId;
    }

    // Mapping from entity types to url entityTypes
    params.pageEntityListType = urlEntityListTypes[params.pageEntityListType];
    params.entityType2 =
        urlEntityTypes[params.entityType2] || urlEntityListTypes[params.entityListType2];
    params.entityType1 =
        urlEntityTypes[params.entityType1] || urlEntityListTypes[params.entityListType1];
    params.pageEntityType = urlEntityTypes[params.pageEntityType];

    const queryString = queryParams
        ? qs.stringify(queryParams, {
              addQueryPrefix: true,
              arrayFormat: 'repeat',
              encodeValuesOnly: true
          })
        : '';

    return generatePath(path, params) + queryString;
}

export function parseURL(match, location) {
    if (!match) return {};
    const params = { ...match.params };

    // Mapping from url to entity types
    if (params.pageEntityListType)
        params.pageEntityListType = getTypeKeyFromParamValue(params.pageEntityListType);
    if (params.entityListType2) {
        params.entityListType2 = getTypeKeyFromParamValue(params.entityListType2, true);
    } else if (params.entityType2) {
        if (isListType(params.entityType2)) {
            params.entityListType2 = getTypeKeyFromParamValue(params.entityType2, true);
            delete params.entityType2;
        } else {
            params.entityType2 = getTypeKeyFromParamValue(params.entityType2);
        }
    }

    if (params.entityListType1) {
        params.entityListType1 = getTypeKeyFromParamValue(params.entityListType1);
    } else if (params.entityType1) {
        if (isListType(params.entityType1)) {
            params.entityListType1 = getTypeKeyFromParamValue(params.entityType1, true);
            delete params.entityType1;
        } else {
            params.entityType1 = getTypeKeyFromParamValue(params.entityType1);
        }
    }
    if (params.pageEntityType)
        params.pageEntityType = getTypeKeyFromParamValue(params.pageEntityType);

    const query =
        location && location.search ? qs.parse(location.search, { ignoreQueryPrefix: true }) : {};

    return {
        params,
        query
    };
}
