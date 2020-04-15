import queryService from 'modules/queryService';

export const violationCategories = ['ALERTS'];
export const processCategories = ['PROCESS'];

const violationQueryPrefix = 'Tag';
const processQueryPrefix = 'Process Tag';

function getQuery(prefix, queryText) {
    const query = queryService.objectToWhereClause({ [prefix]: queryText });
    if (!query) return `${prefix}:`;
    return query;
}

export function getViolationQuery(queryText) {
    return getQuery(violationQueryPrefix, queryText);
}

export function getProcessQuery(queryText) {
    return getQuery(processQueryPrefix, queryText);
}
