import qs from 'qs';
import pageTypes from 'constants/pageTypes';
import { resourceTypes, standardTypes } from 'constants/entityTypes';
import contextTypes from 'constants/contextTypes';
import { generatePath } from 'react-router-dom';
import { nestedCompliancePaths, resourceTypesToUrl, riskPath, secretsPath } from '../routePaths';

function isResource(type) {
    return Object.values(resourceTypes).includes(type);
}

function getEntityTypeKeyFromValue(entityTypeValue) {
    const match = Object.entries(resourceTypesToUrl).find(entry => entry[1] === entityTypeValue);
    return match ? match[0] : null;
}

function getEntityTypeFromMatch(match) {
    if (!match || !match.params || !match.params.entityType) return null;
    return (
        standardTypes[match.params.entityType] || getEntityTypeKeyFromValue(match.params.entityType)
    );
}

function getPath(context, pageType, urlParams) {
    const isResourceType = urlParams.entityType ? isResource(urlParams.entityType) : false;
    const { entityType } = urlParams;
    const pathMap = {
        [contextTypes.COMPLIANCE]: {
            [pageTypes.DASHBOARD]: nestedCompliancePaths.DASHBOARD,
            [pageTypes.ENTITY]: nestedCompliancePaths[entityType],
            [pageTypes.LIST]: nestedCompliancePaths.LIST
        },
        [contextTypes.RISK]: {
            [pageTypes.ENTITY]: riskPath,
            [pageTypes.LIST]: '/main/risk'
        },
        [contextTypes.SECRET]: {
            [pageTypes.ENTITY]: secretsPath,
            [pageTypes.LIST]: '/main/secrets'
        }
    };

    const contextData = pathMap[context];
    if (!contextData) return null;

    const path = contextData[pageType];
    if (!path) return null;

    const params = { ...urlParams };

    // Patching url params for legacy contexts
    if (context === contextTypes.SECRET) {
        params.secretId = params.entityId;
    } else if (context === contextTypes.RISK) {
        params.deploymentId = params.entityId;
    }

    if (isResourceType) {
        params.entityType = resourceTypesToUrl[params.entityType];
        params.listEntityType = resourceTypesToUrl[params.listEntityType];
    }

    return generatePath(path, params);
}

function getContext(match) {
    if (match.url.includes('/compliance')) return contextTypes.COMPLIANCE;
    if (match.url.includes('/risk')) return contextTypes.RISK;
    return null;
}

function getPageType(match) {
    if (match.params.entityId) return pageTypes.ENTITY;
    if (match.params.entityType) return pageTypes.LIST;
    return pageTypes.DASHBOARD;
}

function getParams(match, location) {
    const newParams = { ...match.params };
    newParams.entityType = getEntityTypeFromMatch(match);

    return {
        ...newParams,
        context: getContext(match),
        pageType: getPageType(match),
        query: qs.parse(location.search, { ignoreQueryPrefix: true })
    };
}

function getLinkTo(context, pageType, params) {
    const { query, ...urlParams } = params;
    const pathname = getPath(context, pageType, urlParams);
    const search = query ? qs.stringify(query, { addQueryPrefix: true }) : '';

    return {
        pathname,
        search,
        url: pathname + search
    };
}

export default {
    getParams,
    getLinkTo,
    getEntityTypeKeyFromValue
};
