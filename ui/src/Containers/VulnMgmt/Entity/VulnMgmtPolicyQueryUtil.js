import entityTypes from 'constants/entityTypes';
import queryService from 'modules/queryService';

export const vulMgmtPolicyQuery = {
    policyQuery: queryService.objectToWhereClause({
        Category: 'Vulnerability Management'
    })
};

export function tryUpdateQueryWithVulMgmtPolicyClause(entityType, search) {
    return entityType === entityTypes.POLICY
        ? queryService.objectToWhereClause({ ...search, Category: 'Vulnerability Management' })
        : queryService.objectToWhereClause(search);
}

export function getScopeQuery(entityContext) {
    return queryService.objectToWhereClause({
        ...queryService.entityContextToQueryObject(entityContext)
    });
}
