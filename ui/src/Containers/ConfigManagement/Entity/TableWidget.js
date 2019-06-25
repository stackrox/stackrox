import React, { useState } from 'react';
import Widget from 'Components/Widget';

import TablePagination from 'Components/TablePagination';
import Table from 'Components/Table';

const TableWidget = ({ header, ...rest }) => {
    const [page, setPage] = useState(0);
    const {
        columns,
        rows,
        onRowClick,
        selectedRowId,
        idAttribute,
        noDataText,
        setTableRef,
        trClassName,
        showThead,
        ...widgetProps
    } = { ...rest };
    const headerComponents = (
        <TablePagination page={page} dataLength={rows.length} setPage={setPage} />
    );
    return (
        <Widget header={header} headerComponents={headerComponents} {...widgetProps}>
            <Table
                columns={columns}
                rows={rows}
                onRowClick={onRowClick}
                selectedRowId={selectedRowId}
                idAttribute={idAttribute}
                noDataText={noDataText}
                setTableRef={setTableRef}
                trClassName={trClassName}
                showThead={showThead}
            />
        </Widget>
    );
};

export default TableWidget;
