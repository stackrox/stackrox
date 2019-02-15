import qs from 'qs';
import merge from 'deepmerge';
import queryMap from './queryMap';

function objectToWhereClause(query) {
    if (!query) return '';

    return Object.entries(query)
        .reduce((acc, entry) => {
            const [key, value] = entry;
            if (!value) return acc;
            const flatValue = Array.isArray(value) ? value.join() : value;
            return `${acc}${key}:${flatValue}+`;
        }, '')
        .slice(0, -1);
}

function constructWhereClause(mappedVariables, params) {
    const newlyCreatedMappedVariables = { ...mappedVariables };

    let { query } = params;
    const { groupBy, ...rest } = query;
    if (mappedVariables.where) {
        query = merge(rest, qs.parse(mappedVariables.where));
    }

    newlyCreatedMappedVariables.where = objectToWhereClause(query);
    return newlyCreatedMappedVariables;
}

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

    const { query, variables, format, bypassCache = false } = matches[0].config;
    let mappedVariables = variables.reduce((acc, param) => {
        if (param.graphQLValue) {
            acc[param.graphQLParam] = param.graphQLValue;
        } else if (param.paramsFunc) {
            acc[param.graphQLParam] = param.paramsFunc(params);
        } else {
            const queryParamValue = params[param.queryParam];
            acc[param.graphQLParam] = queryParamValue;
        }
        return acc;
    }, {});

    mappedVariables = constructWhereClause(mappedVariables, params);

    return {
        query,
        variables: mappedVariables,
        format,
        bypassCache
    };
}

export default {
    getQuery,
    objectToWhereClause
};
