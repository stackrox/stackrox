import type { ReactElement } from 'react';
// @ts-expect-error no type definitions available for object-resolve-path
import resolvePath from 'object-resolve-path';

type TableCellValueProps<T> = {
    row: T;
    column: {
        Header: string;
        accessor: ((data: T) => string) | string;
    };
};

function TableCellValue<T>({ row, column }: TableCellValueProps<T>): ReactElement {
    let value: string;
    if (typeof column.accessor === 'function') {
        value = column.accessor(row).toString();
    } else {
        value = resolvePath(row, column.accessor).toString();
    }
    return <div>{value || '-'}</div>;
}

export default TableCellValue;
