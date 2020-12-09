import entityTypes from 'constants/entityTypes';

export function getFilteredCVEColumns(columns, workflowState) {
    const shouldRemoveFixedByColumn =
        !workflowState.isPreceding(entityTypes.COMPONENT) &&
        !workflowState.isCurrentSingle(entityTypes.COMPONENT);

    const shouldRemoveDiscoveredAtImageColumn =
        !workflowState.isPreceding(entityTypes.IMAGE) &&
        !workflowState.isCurrentSingle(entityTypes.IMAGE) &&
        !workflowState.getSingleAncestorOfType(entityTypes.IMAGE);

    return columns.filter((col) => {
        switch (col.accessor) {
            case 'fixedByVersion': {
                if (shouldRemoveFixedByColumn) {
                    return false;
                }
                break;
            }
            case 'discoveredAtImage': {
                if (shouldRemoveDiscoveredAtImageColumn) {
                    return false;
                }
                break;
            }
            default: {
                break;
            }
        }
        return true;
    });
}

export default {
    getFilteredCVEColumns,
};
