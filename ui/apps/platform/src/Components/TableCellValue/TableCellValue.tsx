import React from 'react';
import resolvePath from 'object-resolve-path';

type TableCellProps<T> = {
    row: T;
    column: {
        Header: string;
        accessor: ((data) => string) | string;
    };
};

function TableCellValue<T>({ row, column }: TableCellProps<T>): React.ReactElement {
    let value: string;
    if (typeof column.accessor === 'function') {
        value = column.accessor(row).toString();
    } else {
        value = resolvePath(row, column.accessor).toString();
    }
    return <div>{value || '-'}</div>;
}

export default TableCellValue;
