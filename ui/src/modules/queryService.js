import { singular } from 'pluralize';
import toUpper from 'lodash/toUpper';
import queryMap from './queryMap';

function constructWhereClause(mappedVariables, params) {
    const newlyCreatedMappedVariables = { ...mappedVariables };
    let whereClause = '';

    Object.keys(params.query).forEach((queryParamKey, index) => {
        const queryParamValue = params.query[queryParamKey];
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

    const { query, variables, format } = matches[0].config;
    let mappedVariables = variables.reduce((acc, param) => {
        if (param.graphQLValue) {
            acc[param.graphQLParam] = param.graphQLValue;
        } else {
            let queryParamValue = params[param.queryParam];
            if (param.queryParam === 'entityType') {
                queryParamValue = toUpper(singular(queryParamValue));
            }
            acc[param.graphQLParam] = queryParamValue;
        }
        return acc;
    }, {});

    mappedVariables = constructWhereClause(mappedVariables, params);

    return {
        query,
        variables: mappedVariables,
        format
    };
}

export default {
    getQuery
};
