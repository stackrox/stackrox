import React, { Component, ReactNode } from 'react';

import { ChevronRight, ChevronDown } from 'react-feather';

import { Cell } from './tableTypes';

const expanderColumnId = 'expander';

export function isExpanderCell(cell: Cell): boolean {
    return cell.column.id === expanderColumnId;
}

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
    hooks.visibleColumns.push(
        (visibleColumns) =>
            [
                {
                    // Build our expander column
                    id: expanderColumnId,
                    Cell: ExpanderCellComponent,
                },
                ...visibleColumns,
            ] as { id: string; Cell: Component }[]
    );
}

export default expanderPlugin;
