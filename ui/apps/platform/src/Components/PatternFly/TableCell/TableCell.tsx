import type { ReactElement } from 'react';
import { Td } from '@patternfly/react-table';
import get from 'lodash/get';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function TableCell({ row, column }: { row: any; column: any }): ReactElement {
    let value = get(row, column.accessor);
    if (column.Cell) {
        value = column.Cell({ original: row, value });
    }
    return (
        <Td key={column.Header} dataLabel={column.Header}>
            {value || '-'}
        </Td>
    );
}

export default TableCell;
