import uniq from 'lodash/uniq';

import entityTypes from 'constants/entityTypes';

export function getFilteredCVEColumns(columns, workflowState, isFeatureFlagEnabled) {
    const shouldKeepActiveColumn =
        isFeatureFlagEnabled('ROX_ACTIVE_VULN_MGMT') &&
        (workflowState.isCurrentSingle(entityTypes.DEPLOYMENT) ||
            workflowState.isPrecedingSingle(entityTypes.DEPLOYMENT) ||
            (workflowState.getSingleAncestorOfType(entityTypes.DEPLOYMENT) &&
                workflowState.getSingleAncestorOfType(entityTypes.IMAGE)));

    const shouldKeepFixedByColumn =
        workflowState.isPreceding(entityTypes.IMAGE_COMPONENT) ||
        workflowState.isCurrentSingle(entityTypes.IMAGE_COMPONENT) ||
        workflowState.isPreceding(entityTypes.NODE_COMPONENT) ||
        workflowState.isCurrentSingle(entityTypes.NODE_COMPONENT);

    const shouldKeepDiscoveredAtImageColumn =
        workflowState.isPreceding(entityTypes.IMAGE) ||
        workflowState.isCurrentSingle(entityTypes.IMAGE) ||
        workflowState.getSingleAncestorOfType(entityTypes.IMAGE);

    const currentEntityType = workflowState.getCurrentEntityType();

    const shouldKeepDiscoveredTime = currentEntityType !== entityTypes.CLUSTER_CVE;

    // No need to show entities in the node component or cluster context.
    const shouldKeepEntitiesColumn =
        !workflowState.isPrecedingSingle(entityTypes.NODE_COMPONENT) ||
        !workflowState.getSingleAncestorOfType(entityTypes.NODE);
    // special case CLUSTER CVE under CLUSTER
    const clusterCveUnderCluster =
        workflowState.getSingleAncestorOfType(entityTypes.CLUSTER) &&
        currentEntityType === entityTypes.CLUSTER_CVE;

    const shouldKeepCveType = currentEntityType === entityTypes.CLUSTER_CVE;

    const shouldKeepSeverity =
        currentEntityType === entityTypes.IMAGE_CVE || currentEntityType === entityTypes.NODE_CVE;

    return columns.filter((col) => {
        switch (col.accessor) {
            case 'vulnerabilityTypes': {
                return !!shouldKeepCveType;
            }
            case 'isActive': {
                return !!shouldKeepActiveColumn;
            }
            case 'fixedByVersion': {
                return shouldKeepFixedByColumn;
            }
            case 'createdAt': {
                return shouldKeepDiscoveredTime;
            }
            case 'discoveredAtImage': {
                return shouldKeepDiscoveredAtImageColumn;
            }
            case 'entities': {
                return shouldKeepEntitiesColumn && !clusterCveUnderCluster;
            }
            case 'severity': {
                return shouldKeepSeverity || shouldKeepDiscoveredAtImageColumn;
            }
            default: {
                return true;
            }
        }
    });
}

export function parseCveNamesFromIds(cveIds) {
    const cveNames = cveIds.map((cveId) => {
        return cveId.split('#')[0];
    });

    return uniq(cveNames);
}

export default {
    getFilteredCVEColumns,
    parseCveNamesFromIds,
};
