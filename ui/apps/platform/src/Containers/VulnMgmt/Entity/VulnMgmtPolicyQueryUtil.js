import entityTypes from 'constants/entityTypes';
import queryService from 'utils/queryService';
import { withActiveDeploymentQuery } from 'utils/deploymentUtils';

export const vulMgmtPolicyQuery = {
    policyQuery: queryService.objectToWhereClause({
        Category: 'Vulnerability Management',
    }),
};

export function tryUpdateQueryWithVulMgmtPolicyClause(
    entityType,
    search,
    entityContext,
    isDeploymentSoftDeletionEnabled = false
) {
    const query =
        entityType === entityTypes.POLICY
            ? queryService.objectToWhereClause({ ...search, Category: 'Vulnerability Management' })
            : queryService.objectToWhereClause(search);
    // When listing deployments, filter out soft-deleted ones.
    if (entityType === entityTypes.DEPLOYMENT) {
        return withActiveDeploymentQuery(query, isDeploymentSoftDeletionEnabled);
    }
    return query;
}

export function getScopeQuery(entityContext) {
    return queryService.objectToWhereClause({
        ...queryService.entityContextToQueryObject(entityContext),
    });
}
