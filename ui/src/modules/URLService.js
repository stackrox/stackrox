import qs from 'qs';
import pageTypes from 'constants/pageTypes';
import { resourceTypes } from 'constants/entityTypes';
import contextTypes from 'constants/contextTypes';
import { generatePath } from 'react-router-dom';
import { nestedCompliancePaths } from '../routePaths';

function getPath(context, pageType, urlParams) {
    const isResource = Object.values(resourceTypes).includes(urlParams.entityType);
    const pathMap = {
        [contextTypes.COMPLIANCE]: {
            [pageTypes.DASHBOARD]: nestedCompliancePaths.DASHBOARD,
            [pageTypes.ENTITY]: isResource
                ? nestedCompliancePaths.RESOURCE
                : nestedCompliancePaths.CONTROL,
            [pageTypes.LIST]: nestedCompliancePaths.LIST
        }
    };

    const contextData = pathMap[context];
    if (!contextData) return null;

    const path = contextData[pageType];
    if (!path) return null;

    return generatePath(path, urlParams);
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
    return {
        ...match.params,
        context: getContext(match),
        pageType: getPageType(match),
        query: qs.parse(location.search, { ignoreQueryPrefix: true })
    };
}

function getLinkTo(context, pageType, params) {
    const { query, ...urlParams } = params;
    return {
        pathname: getPath(context, pageType, urlParams),
        search: query ? qs.stringify(query, { addQueryPrefix: true }) : null
    };
}

export default {
    getParams,
    getLinkTo
};
