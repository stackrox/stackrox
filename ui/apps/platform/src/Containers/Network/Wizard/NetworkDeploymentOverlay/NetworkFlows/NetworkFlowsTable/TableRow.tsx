import React, { ReactElement, ReactNode } from 'react';

export type TableRowProps = {
    row: {
        getRowProps: () => {
            key: string;
        };
    };
    children: ReactNode;
};

function TableRow({ row, children }: TableRowProps): ReactElement {
    const { key } = row.getRowProps();

    return (
        <tr key={key} className="border-b border-base-300">
            {children}
        </tr>
    );
}

export default TableRow;
