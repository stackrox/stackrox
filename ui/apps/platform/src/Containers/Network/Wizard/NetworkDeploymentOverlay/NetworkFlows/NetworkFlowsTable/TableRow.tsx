import React, { ReactElement, ReactNode } from 'react';

export type TableRowProps = {
    type: 'alert' | null;
    row: {
        getRowProps: () => {
            key: string;
        };
    };
    children: ReactNode;
};

const tableRowClassName = 'border-b';
const baseTableRowClassName = `${tableRowClassName} border-base-300`;
const alertTableRowClassName = `${tableRowClassName} border-alert-300 bg-alert-200 text-alert-800`;

function TableRow({ type, row, children }: TableRowProps): ReactElement {
    const { key } = row.getRowProps();
    const className = type === 'alert' ? alertTableRowClassName : baseTableRowClassName;

    return (
        <tr key={key} className={className}>
            {children}
        </tr>
    );
}

export default TableRow;
