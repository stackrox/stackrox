import entityTypes from 'constants/entityTypes';

export function getFilteredComponentColumns(columns, workflowState) {
    const shouldRemoveColumns =
        workflowState.isBaseList(entityTypes.COMPONENT) ||
        workflowState.isChildOfEntity(entityTypes.CVE) ||
        workflowState.isChildOfEntity(entityTypes.DEPLOYMENT);

    return shouldRemoveColumns
        ? columns.filter(col => col.accessor !== 'source' && col.accessor !== 'location')
        : columns;
}

export default {
    getFilteredComponentColumns
};
