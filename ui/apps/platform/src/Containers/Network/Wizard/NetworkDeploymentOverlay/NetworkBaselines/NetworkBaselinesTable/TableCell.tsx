import React, { ReactElement } from 'react';

export type TableCellProps = {
    cell: {
        getCellProps: () => {
            key: string;
        };
        row: {
            isGrouped: boolean;
        };
        column: {
            id: string;
        };
        render: (string) => ReactElement;
    };
};

function TableCell({ cell }: TableCellProps): ReactElement {
    const { key } = cell.getCellProps();

    const className = `text-left p-2 ${
        !cell.row.isGrouped &&
        cell.column.id === 'expander' &&
        'bg-primary-200 border-r border-primary-300'
    }`;

    return (
        <td key={key} className={className} data-testid="data-cell">
            {cell.render('Cell')}
        </td>
    );
}

export default TableCell;
