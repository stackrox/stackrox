import entityTypes from 'constants/entityTypes';

export function getFilteredCVEColumns(columns, workflowState) {
    // TODO: when active status for CVEs becomes available
    // uncomment the following check
    // const shouldKeepActiveColumn =
    //     workflowState.isCurrentSingle(entityTypes.DEPLOYMENT) ||
    //     workflowState.isPrecedingSingle(entityTypes.DEPLOYMENT);

    const shouldKeepFixedByColumn =
        workflowState.isPreceding(entityTypes.COMPONENT) ||
        workflowState.isCurrentSingle(entityTypes.COMPONENT);

    const shouldKeepDiscoveredAtImageColumn =
        workflowState.isPreceding(entityTypes.IMAGE) ||
        workflowState.isCurrentSingle(entityTypes.IMAGE) ||
        workflowState.getSingleAncestorOfType(entityTypes.IMAGE);

    // No need to show entities in the node component context.
    const shouldKeepEntitiesColumn =
        !workflowState.isPrecedingSingle(entityTypes.COMPONENT) ||
        !workflowState.getSingleAncestorOfType(entityTypes.NODE);

    return columns.filter((col) => {
        switch (col.accessor) {
            case 'isActive': {
                // TODO: when active status for CVEs becomes available
                // uncomment the following actual check, and remove the always-false return
                // return !!shouldKeepActiveColumn;
                return false;
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
