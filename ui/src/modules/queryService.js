import queryMap from './queryMap';

function getQuery(pageId, params, component) {
    const { context, pageType, entityType } = pageId;

    const config = queryMap.filter(
        item =>
            (!item.context.length || item.context.includes(context)) &&
            (!item.pageType.length || item.pageType.includes(pageType)) &&
            (!item.entityType.length || item.entityType.includes(entityType)) &&
            (!item.component.length || item.component.includes(component))
    );

    if (config.length === 0) return null;

    if (config.length > 1)
        throw Error(
            `More than one query matching ${context}, ${pageType}, ${entityType}, ${component}`
        );

    const { query, variables: queryVars } = config[0];
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
