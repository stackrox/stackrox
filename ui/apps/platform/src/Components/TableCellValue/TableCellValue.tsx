import React from 'react';
import resolvePath from 'object-resolve-path';
import { Integration } from '../../Containers/Integrations/utils/integrationUtils';
import { SignatureIntegration } from '../../services/SignatureIntegrationsService';

type TableCellProps = {
    row: Integration | SignatureIntegration;
    column: {
        Header: string;
        accessor: ((data) => string) | string;
    };
};

function TableCellValue({ row, column }: TableCellProps): React.ReactElement {
    let value: string;
    if (typeof column.accessor === 'function') {
        value = column.accessor(row).toString();
    } else {
        value = resolvePath(row, column.accessor).toString();
    }
    return <div>{value || '-'}</div>;
}

export default TableCellValue;
