import entityTypes from 'constants/entityTypes';

export function getFilteredComponentColumns(columns, workflowState) {
    const shouldRemoveColumns = !workflowState.isPreceding(entityTypes.IMAGE);

    return columns.filter((col) => {
        switch (col.accessor) {
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
