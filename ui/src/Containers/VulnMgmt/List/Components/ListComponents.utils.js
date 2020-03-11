import entityTypes from 'constants/entityTypes';

export function getFilteredComponentColumns(columns, workflowState) {
    const shouldRemoveColumns = !workflowState.isPreceding(entityTypes.IMAGE);

    return shouldRemoveColumns
        ? columns.filter(col => col.accessor !== 'source' && col.accessor !== 'location')
        : columns;
}

export default {
    getFilteredComponentColumns
};
