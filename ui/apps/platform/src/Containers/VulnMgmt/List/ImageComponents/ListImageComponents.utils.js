import entityTypes from 'constants/entityTypes';

export function getFilteredComponentColumns(columns, workflowState) {
    const shouldKeepActiveColumn =
        workflowState.isCurrentSingle(entityTypes.DEPLOYMENT) ||
        workflowState.isPrecedingSingle(entityTypes.DEPLOYMENT) ||
        (workflowState.getSingleAncestorOfType(entityTypes.DEPLOYMENT) &&
            workflowState.getSingleAncestorOfType(entityTypes.IMAGE));

    const shouldRemoveColumns = !workflowState.isPreceding(entityTypes.IMAGE);

    return columns.filter((col) => {
        switch (col.accessor) {
            case 'isActive': {
                return !!shouldKeepActiveColumn;
            }
            case 'source': {
                return !shouldRemoveColumns;
            }
            case 'location': {
                return !shouldRemoveColumns;
            }
            default: {
                return true;
            }
        }
    });
}

export default {
    getFilteredComponentColumns,
};
