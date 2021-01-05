import React, { ReactNode } from 'react';

import IndeterminateCheckbox from './IndeterminateCheckbox';

function CheckboxCellComponent({ row }): ReactNode {
    const { title, checked, indeterminate, onChange } = row.getToggleRowSelectedProps();
    const { onClick: toggleExpand } = row.getToggleRowExpandedProps();

    function onChangeHandler(event: React.ChangeEvent): void {
        onChange(event);
        // If a row has nested rows, and was checked, expanded it if it isn't already
        if (!checked && !row.isExpanded && row.subRows.length > 1) {
            toggleExpand();
        }
    }

    return (
        <IndeterminateCheckbox
            title={title}
            checked={checked}
            indeterminate={indeterminate}
            onChange={onChangeHandler}
        />
    );
}

export type CheckboxSelectionPluginOptions = {
    showHeader: boolean;
};

function checkboxSelectionPlugin(hooks): void {
    hooks.visibleColumns.push((visibleColumns) => [
        // Make a column for selection
        {
            id: 'selection',
            Cell: CheckboxCellComponent,
        },
        ...visibleColumns,
    ]);
}

export default checkboxSelectionPlugin;
