import React, { useState } from 'react';
import { withRouter } from 'react-router-dom';
import resolvePath from 'object-resolve-path';

import Widget from 'Components/Widget';
import TablePagination from 'Components/TablePagination';
import Table from 'Components/Table';
import URLService from 'modules/URLService';

const TableWidget = ({ match, location, history, header, entityType, ...rest }) => {
    const [page, setPage] = useState(0);
    const {
        columns,
        rows,
        selectedRowId,
        idAttribute,
        noDataText,
        setTableRef,
        trClassName,
        showThead,
        SubComponent,
        hasNestedTable,
        ...widgetProps
    } = { ...rest };
    const headerComponents = (
        <TablePagination page={page} dataLength={rows.length} setPage={setPage} />
    );
    function onRowClick(row) {
        const id = resolvePath(row, idAttribute);
        const url = URLService.getURL(match, location)
            .push(entityType, id)
            .url();
        history.push(url);
    }
    return (
        <Widget
            header={header}
            headerComponents={headerComponents}
            {...widgetProps}
            className="w-full"
        >
            <Table
                columns={columns}
                rows={rows}
                // If "SubComponent" exists, clicking on the row should do nothing because the expander is used
                onRowClick={SubComponent || hasNestedTable ? null : onRowClick}
                selectedRowId={selectedRowId}
                idAttribute={idAttribute}
                noDataText={noDataText}
                setTableRef={setTableRef}
                trClassName={trClassName}
                showThead={showThead}
                SubComponent={SubComponent}
                page={page}
            />
        </Widget>
    );
};

export default withRouter(TableWidget);
