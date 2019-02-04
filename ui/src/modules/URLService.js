import qs from 'qs';
import pageTypes from 'constants/pageTypes';
import { resourceTypes } from 'constants/entityTypes';
import contextTypes from 'constants/contextTypes';
import { generatePath } from 'react-router-dom';
import { nestedCompliancePaths, resourceTypesToUrl } from '../routePaths';

function isResource(type) {
    return Object.values(resourceTypes).includes(type);
}

function getResourceTypeFromMatch(match) {
    if (!match || !match.params || !match.params.entityType) return null;

    const entityEntry = Object.entries(resourceTypesToUrl).find(
        entry => entry[1] === match.params.entityType
    );
    if (!entityEntry) return null;

    return entityEntry[0];
}
function getPath(context, pageType, urlParams) {
    const isResourceType = urlParams.entityType ? isResource(urlParams.entityType) : false;
    const pathMap = {
        [contextTypes.COMPLIANCE]: {
            [pageTypes.DASHBOARD]: nestedCompliancePaths.DASHBOARD,
            [pageTypes.ENTITY]: isResourceType
                ? nestedCompliancePaths.RESOURCE
                : nestedCompliancePaths.CONTROL,
            [pageTypes.LIST]: nestedCompliancePaths.LIST
        }
    };

    const contextData = pathMap[context];
    if (!contextData) return null;

    const path = contextData[pageType];
    if (!path) return null;

    const params = { ...urlParams };
    if (isResourceType) {
        params.entityType = resourceTypesToUrl[urlParams.entityType];
    }
    return generatePath(path, params);
}

function getContext(match) {
    if (match.url.includes('/compliance')) return contextTypes.COMPLIANCE;
    return null;
}

function getPageType(match) {
    if (match.params.entityId) return pageTypes.ENTITY;
    if (match.params.entityType) return pageTypes.LIST;
    return pageTypes.DASHBOARD;
}

function getParams(match, location) {
    const newParams = { ...match.params };
    newParams.entityType = getResourceTypeFromMatch(match);

    return {
        ...match.params,
        context: getContext(match),
        pageType: getPageType(match),
        query: qs.parse(location.search, { ignoreQueryPrefix: true })
    };
}

function getLinkTo(context, pageType, params) {
    const { query, ...urlParams } = params;
    const pathname = getPath(context, pageType, urlParams);
    const search = query ? qs.stringify(query, { addQueryPrefix: true }) : null;
    return {
        pathname,
        search,
        url: pathname + search
    };
}

export default {
    getParams,
    getLinkTo
};
