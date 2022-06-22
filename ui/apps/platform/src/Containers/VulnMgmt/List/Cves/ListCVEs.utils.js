import entityTypes from 'constants/entityTypes';

export function getFilteredCVEColumns(columns, workflowState) {
    const shouldKeepActiveColumn =
        workflowState.isCurrentSingle(entityTypes.DEPLOYMENT) ||
        workflowState.isPrecedingSingle(entityTypes.DEPLOYMENT) ||
        (workflowState.getSingleAncestorOfType(entityTypes.DEPLOYMENT) &&
            workflowState.getSingleAncestorOfType(entityTypes.IMAGE));

    const shouldKeepFixedByColumn =
        workflowState.isPreceding(entityTypes.COMPONENT) ||
        workflowState.isCurrentSingle(entityTypes.COMPONENT);

    const shouldKeepDiscoveredAtImageColumn =
        workflowState.isPreceding(entityTypes.IMAGE) ||
        workflowState.isCurrentSingle(entityTypes.IMAGE) ||
        workflowState.getSingleAncestorOfType(entityTypes.IMAGE);

    const currentEntityType = workflowState.getCurrentEntityType();

    // No need to show entities in the node component context.
    const shouldKeepEntitiesColumn =
        (!workflowState.isPrecedingSingle(entityTypes.COMPONENT) ||
            !workflowState.getSingleAncestorOfType(entityTypes.NODE)) &&
        currentEntityType !== entityTypes.CLUSTER_CVE;

    // TODO: remove this temporary conditional check, after generic CVE list is removed
    const shouldKeepCveType = currentEntityType === entityTypes.CVE;

    return columns.filter((col) => {
        switch (col.accessor) {
            // TODO: remove after generic CVE list is removed
            case 'vulnerabilityTypes': {
                return !!shouldKeepCveType;
            }
            case 'isActive': {
                return !!shouldKeepActiveColumn;
            }
            case 'fixedByVersion': {
                return shouldKeepFixedByColumn;
            }
            case 'discoveredAtImage': {
                return shouldKeepDiscoveredAtImageColumn;
            }
            case 'entities': {
                return shouldKeepEntitiesColumn;
            }
            case 'severity': {
                return shouldKeepDiscoveredAtImageColumn;
            }
            default: {
                return true;
            }
        }
    });
}

export default {
    getFilteredCVEColumns,
};
