import React, { ReactElement } from 'react';

export type TableCellProps = {
    cell: {
        getCellProps: () => {
            key: string;
        };
        render: (string) => ReactElement;
    };
};

function TableCell({ cell }: TableCellProps): ReactElement {
    const { key } = cell.getCellProps();

    return (
        <td key={key} className="text-left p-2">
            {cell.render('Cell')}
        </td>
    );
}

export default TableCell;
