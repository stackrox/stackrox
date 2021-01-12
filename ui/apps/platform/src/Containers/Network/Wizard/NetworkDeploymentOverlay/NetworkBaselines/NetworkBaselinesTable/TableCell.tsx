import React, { ReactElement } from 'react';

import { Cell } from './tableTypes';
import { isExpanderCell } from './expanderPlugin';

export type TableCellProps = {
    cell: Cell;
};

function TableCell({ cell }: TableCellProps): ReactElement {
    const { key } = cell.getCellProps();

    const className = `text-left p-2 ${
        !cell.row.isGrouped && isExpanderCell(cell) && 'bg-primary-200 border-r border-primary-300'
    }`;

    return (
        <td key={key} className={className} data-testid="data-cell">
            {cell.render('Cell')}
        </td>
    );
}

export default TableCell;
