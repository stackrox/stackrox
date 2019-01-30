import queryMap from './queryMap';

function getQuery(params, component) {
    const { context, pageType, entityType } = params;

    const matches = queryMap.filter(
        item =>
            (!item.context.length || item.context.includes(context)) &&
            (!item.pageType.length || item.pageType.includes(pageType)) &&
            (!item.entityType.length || item.entityType.includes(entityType)) &&
            (!item.component.length || item.component.includes(component))
    );

    if (matches.length === 0) return null;

    if (matches.length > 1)
        throw Error(
            `More than one query matching ${context}, ${pageType}, ${entityType}, ${component}`
        );

    const { query, variables, format } = matches[0].config;
    const mappedVariables = variables.reduce((acc, param) => {
        if (param.graphQLValue) {
            acc[param.graphQLParam] = param.graphQLValue;
        } else {
            acc[param.graphQLParam] = params[param.queryParam];
        }
        return acc;
    }, {});

    return {
        query,
        variables: mappedVariables,
        format
    };
}

export default {
    getQuery
};
