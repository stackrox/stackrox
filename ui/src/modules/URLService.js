import qs from 'qs';
import pageTypes from 'constants/pageTypes';
import contextTypes from 'constants/contextTypes';

function getContext(match) {
    if (match.url.includes('/compliance')) return contextTypes.COMPLIANCE;
    return null;
}

function getPageType(match) {
    if (match.params.entityId) return pageTypes.ENTITY;
    if (match.params.entityType) return pageTypes.LIST;
    return pageTypes.DASHBOARD;
}

function getEntityType(match) {
    return match.params.entityType;
}

function getPageId(match) {
    const context = getContext(match);
    const pageType = getPageType(match);
    const entityType = getEntityType(match);
    return {
        context,
        pageType,
        entityType
    };
}

function getParams(match, location) {
    return {
        ...match.params,
        query: qs.parse(location.search, { ignoreQueryPrefix: true })
    };
}

export default {
    getParams,
    getPageId
};
