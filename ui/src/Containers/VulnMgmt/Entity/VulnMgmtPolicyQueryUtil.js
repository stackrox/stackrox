import entityTypes from 'constants/entityTypes';
import queryService from 'modules/queryService';

const entitiesWithPolicyField = [
    entityTypes.DEPLOYMENT,
    entityTypes.CLUSTER,
    entityTypes.NAMESPACE
];

export function getPolicyQueryVar(entityType) {
    return entitiesWithPolicyField.includes(entityType) ? ', $policyQuery: String' : '';
}

export function tryUpdateQueryWithVulMgmtPolicyClause(entityType, search) {
    return entityType === entityTypes.POLICY
        ? queryService.objectToWhereClause({ ...search, Category: 'Vulnerability Management' })
        : queryService.objectToWhereClause(search);
}
