import React, { ReactNode } from 'react';

import { ChevronRight, ChevronDown } from 'react-feather';

function ExpanderCellComponent({ row }): ReactNode {
    if (!row.canExpand || row.subRows.length <= 1) {
        return null;
    }
    const { onClick } = row.getToggleRowExpandedProps({});

    return (
        <button type="button" onClick={onClick}>
            {row.isExpanded ? (
                <ChevronDown className="h-4 w-4" />
            ) : (
                <ChevronRight className="h-4 w-4" />
            )}
        </button>
    );
}

function expanderPlugin(hooks): void {
    hooks.visibleColumns.push((visibleColumns) => [
        {
            // Build our expander column
            id: 'expander', // Make sure it has an ID
            Cell: ExpanderCellComponent,
        },
        ...visibleColumns,
    ]);
}

export default expanderPlugin;
