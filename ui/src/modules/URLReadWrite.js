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
              arrayFormat: 'repeat',
              encodeValuesOnly: true
          })
        : '';

    return generatePath(path, params) + queryString;
}

// Convert URL to workflow state and search objects
export function parseURL(match, location) {
    if (!match) return {};
    const params = { ...match.params };
    const query =
        location && location.search ? qs.parse(location.search, { ignoreQueryPrefix: true }) : {};
    const { workflowState: stateStack = {}, ...searchState } = query;
    const workflowState = { stateStack, useCase: params.context };

    // Convert URL parameter values to enum types
    if (params.pageEntityListType) {
        stateStack.unshift({ t: getTypeKeyFromParamValue(params.pageEntityListType) });
    } else if (params.pageEntityType) {
        stateStack.unshift({
            t: getTypeKeyFromParamValue(params.pageEntityType),
            i: params.pageEntityId
        });
    }
    return {
        workflowState,
        searchState
    };
}
