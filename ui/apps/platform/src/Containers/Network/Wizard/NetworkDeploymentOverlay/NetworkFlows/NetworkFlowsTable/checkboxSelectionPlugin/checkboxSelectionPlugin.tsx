import React, { ReactNode } from 'react';

import IndeterminateCheckbox from './IndeterminateCheckbox';

function CheckboxCellComponent({ row }): ReactNode {
    const { title, checked, indeterminate, onChange } = row.getToggleRowSelectedProps();
    return (
        <IndeterminateCheckbox
            title={title}
            checked={checked}
            indeterminate={indeterminate}
            onChange={onChange}
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
