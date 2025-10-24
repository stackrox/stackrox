import type { ReactElement } from 'react';
import { Td } from '@patternfly/react-table';
import get from 'lodash/get';

function TableCell({ row, column }): ReactElement {
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
