import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { useLocation, useNavigate, useMatch } from 'react-router-dom';
import resolvePath from 'object-resolve-path';

import Widget from 'Components/Widget';
import TablePagination from 'Components/TablePagination';
import Table from 'Components/Table';
import URLService from 'utils/URLService';
import { workflowPaths } from 'routePaths';

const TableWidget = ({ header, entityType, ...rest }) => {
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
        defaultSorted,
        ...widgetProps
    } = { ...rest };

    const navigate = useNavigate();
    const location = useLocation();
    const match = useMatch(workflowPaths.LIST);

    const headerComponents = (
        <TablePagination page={page} dataLength={rows.length} setPage={setPage} />
    );
    function onRowClick(row) {
        const id = resolvePath(row, idAttribute);
        const url = URLService.getURL(match, location).push(entityType, id).url();
        navigate(url);
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
                defaultSorted={defaultSorted}
            />
        </Widget>
    );
};

TableWidget.propTypes = {
    header: PropTypes.oneOfType([PropTypes.element, PropTypes.string]).isRequired,
    entityType: PropTypes.string,
};

TableWidget.defaultProps = {
    entityType: '',
};
export default TableWidget;
