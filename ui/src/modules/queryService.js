import pageTypes from 'constants/pageTypes';
import contextTypes from 'constants/contextTypes';
import URLService from './URLService';
import queryMap from './queryMap';

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

function getQuery(match, location, component) {
    const context = getContext(match);
    const pageType = getPageType(match);
    const entityType = getEntityType(match);

    const config = queryMap.filter(
        item =>
            item.context.includes(context) &&
            item.pageType.includes(pageType) &&
            item.entityType.includes(entityType) &&
            item.component.includes(component)
    );

    if (config.length === 0) return null;

    if (config.length > 1)
        throw Error(
            `More than one query matching ${context}, ${pageType}, ${entityType}, ${component}`
        );

    const { query, variables: queryVars } = config[0];
    const params = new URLService(match, location).getParams(true);
    const variables = queryVars.reduce((acc, param) => {
        acc[param.graphQLParam] = params[param.queryParam];
        return acc;
    }, {});
    const { metadata } = config[0];

    return {
        query,
        variables,
        metadata
    };
}

export default {
    getQuery
};
