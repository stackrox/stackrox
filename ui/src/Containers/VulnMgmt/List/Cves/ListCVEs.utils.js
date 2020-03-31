import entityTypes from 'constants/entityTypes';

export function getFilteredCVEColumns(columns, workflowState) {
    const shouldRemoveColumns =
        !workflowState.isPreceding(entityTypes.COMPONENT) &&
        !workflowState.isCurrentSingle(entityTypes.COMPONENT);
    return shouldRemoveColumns ? columns.filter(col => col.accessor !== 'fixedByVersion') : columns;
}

export default {
    getFilteredCVEColumns
};
