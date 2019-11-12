import React, { useState, useContext } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import resolvePath from 'object-resolve-path';

import workflowStateContext from 'Containers/workflowStateContext';
import Widget from 'Components/Widget';
import NoResultsMessage from 'Components/NoResultsMessage';
import TablePagination from 'Components/TablePagination';
import Table from 'Components/Table';

const TableWidget = ({ history, header, entityType, ...rest }) => {
    const workflowState = useContext(workflowStateContext);
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
    const headerComponents = (
        <TablePagination page={page} dataLength={rows.length} setPage={setPage} />
    );
    function onRowClick(row) {
        const id = resolvePath(row, idAttribute);
        const url = workflowState.pushRelatedEntity(entityType, id).toUrl();
        history.push(url);
    }
    return (
        <>
            {rows.length ? (
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
            ) : (
                <NoResultsMessage message={noDataText} className="p-6" icon="info" />
            )}
        </>
    );
};

TableWidget.propTypes = {
    header: PropTypes.oneOfType([PropTypes.element, PropTypes.string]).isRequired,
    history: ReactRouterPropTypes.history.isRequired,
    idAttribute: PropTypes.string,
    entityType: PropTypes.string.isRequired
};

TableWidget.defaultProps = {
    idAttribute: 'id'
};
export default withRouter(TableWidget);
