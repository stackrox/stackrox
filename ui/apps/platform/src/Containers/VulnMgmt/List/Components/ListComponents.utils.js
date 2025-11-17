import entityTypes from 'constants/entityTypes';

// eslint-disable-next-line no-unused-vars
export function getFilteredComponentColumns(columns, workflowState, _isFeatureFlagEnabled) {
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
