import React, { ReactElement } from 'react';

import { Cell } from './tableTypes';
import { isExpanderCell } from './expanderPlugin';

const tableCellClassName = 'text-left p-2';
const baseTableCellClassName = `${tableCellClassName} border-b border-base-300 bg-base-100`;
const alertTableCellClassName = `${tableCellClassName} border-b border-alert-300 bg-alert-200`;
const nestedExpanderCellClassName = `${tableCellClassName} bg-primary-200 border-r border-primary-300`;

export type TableCellProps = {
    cell: Cell;
    colorType: 'alert' | null;
    isSticky?: boolean;
};

function TableCell({ cell, colorType, isSticky = false }: TableCellProps): ReactElement {
    const { key } = cell.getCellProps();

    const isSubRowExpanderCell = !cell.row.isGrouped && isExpanderCell(cell);

    let className = isSticky ? 'sticky z-1 top-8 border-t' : '';
    if (isSubRowExpanderCell) {
        className = `${className} ${nestedExpanderCellClassName}`;
    } else if (colorType === 'alert') {
        className = `${className} ${alertTableCellClassName}`;
    } else {
        className = `${className} ${baseTableCellClassName}`;
    }

    return (
        <td key={key} className={className} data-testid="data-cell">
            {cell.render('Cell')}
        </td>
    );
}

export default TableCell;
