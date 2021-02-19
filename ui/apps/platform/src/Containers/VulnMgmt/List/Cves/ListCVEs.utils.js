import entityTypes from 'constants/entityTypes';

export function getFilteredCVEColumns(columns, workflowState) {
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
            case 'fixedByVersion': {
                return shouldKeepFixedByColumn;
            }
            case 'discoveredAtImage': {
                return shouldKeepDiscoveredAtImageColumn;
            }
            case 'entities': {
                return shouldKeepEntitiesColumn;
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
