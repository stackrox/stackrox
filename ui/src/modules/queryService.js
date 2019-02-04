import qs from 'qs';
import merge from 'deepmerge';
import queryMap from './queryMap';

function constructWhereClause(mappedVariables, params) {
    const newlyCreatedMappedVariables = { ...mappedVariables };
    let whereClause = '';

    let { query } = params;
    if (mappedVariables.where) {
        query = merge(query, qs.parse(mappedVariables.where));
    }

    Object.keys(query).forEach((queryParamKey, index) => {
        const queryParamValue = query[queryParamKey];
        if (Array.isArray(queryParamValue)) {
            whereClause = `${whereClause}${
                index !== 0 ? '+' : ''
            }${queryParamKey}:${queryParamValue.join(',')}`;
        } else {
            whereClause = `${whereClause}${
                index !== 0 ? '+' : ''
            }${queryParamKey}:${queryParamValue}`;
        }
    });

    newlyCreatedMappedVariables.where = whereClause;

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
    getQuery
};
