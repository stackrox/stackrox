import React, { ReactElement } from 'react';

import { Cell, TableColorStyles } from './tableTypes';
import { isExpanderCell } from './expanderPlugin';

export type TableCellProps = {
    cell: Cell;
    colorStyles: TableColorStyles;
    isSticky?: boolean;
};

export function TableCell({ cell, colorStyles, isSticky = false }: TableCellProps): ReactElement {
    const isSubRowExpanderCell = !cell.row.isGrouped && isExpanderCell(cell);
    const { bgColor, borderColor } = colorStyles;

    const tableCellClassName = 'text-left p-2';
    const typedTableCellClassName = `${tableCellClassName} border-b ${borderColor} ${bgColor}`;
    const nestedExpanderCellClassName = `${tableCellClassName} bg-primary-200 border-r border-primary-300`;

    let className = isSticky ? 'sticky z-1 top-8 border-t' : '';
    if (isSubRowExpanderCell) {
        className = `${className} ${nestedExpanderCellClassName}`;
    } else {
        className = `${className} ${typedTableCellClassName}`;
    }

    return (
        <td className={className} data-testid="data-cell">
            {cell.render('Cell')}
        </td>
    );
}
