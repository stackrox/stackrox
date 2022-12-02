import React, { useState, useContext } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import resolvePath from 'object-resolve-path';

import workflowStateContext from 'Containers/workflowStateContext';
import Widget from 'Components/Widget';
import NoResultsMessage from 'Components/NoResultsMessage';
import TablePagination from 'Components/TablePagination';
import Table, { DEFAULT_PAGE_SIZE } from 'Components/Table';

const TableWidget = ({
    history,
    header,
    entityType,
    pageSize,
    parentPageState,
    currentSort,
    sortHandler,
    ...rest
}) => {
    const workflowState = useContext(workflowStateContext);
    const [localPage, setLocalPage] = useState(0);
    const {
        columns,
        rows,
        idAttribute,
        noDataText,
        setTableRef,
        trClassName,
        showThead,
        SubComponent,
        hasNestedTable,
        defaultSorted,
        className,
        headerActions,
        ...widgetProps
    } = { ...rest };

    // extend this component to handler server-side pagination
    const currentPage = parentPageState?.page || localPage;
    const currentPageHandler = parentPageState?.setPage || setLocalPage;
    const totalCount = parentPageState?.totalCount || rows.length;
    const useServerSidePagination = !!sortHandler;

    console.log('table pagination', pageSize, currentPage, totalCount);

    const headerComponents = (
        <div className="flex">
            {headerActions}
            <TablePagination
                pageSize={pageSize}
                page={currentPage}
                dataLength={totalCount}
                setPage={currentPageHandler}
            />
        </div>
    );
    function onRowClick(row) {
        const id = resolvePath(row, idAttribute);
        const url = workflowState.pushRelatedEntity(entityType, id).toUrl();
        history.push(url);
    }

    return (
        <>
            {totalCount ? (
                <Widget
                    header={header}
                    headerComponents={headerComponents}
                    className={`w-full ${className}`}
                    {...widgetProps}
                >
                    <Table
                        columns={columns}
                        rows={rows}
                        // If "SubComponent" exists, clicking on the row should do nothing because the expander is used
                        onRowClick={SubComponent || hasNestedTable ? null : onRowClick}
                        idAttribute={idAttribute}
                        noDataText={noDataText}
                        setTableRef={setTableRef}
                        trClassName={trClassName}
                        showThead={showThead}
                        SubComponent={SubComponent}
                        page={currentPage}
                        defaultSorted={defaultSorted}
                        sorted={currentSort || undefined}
                        manual={useServerSidePagination}
                        onSortedChange={sortHandler}
                        disableSortRemove={useServerSidePagination}
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
    pageSize: PropTypes.number,
    parentPageState: PropTypes.shape({
        page: PropTypes.number,
        setPage: PropTypes.func,
        totalCount: PropTypes.number,
    }),
    currentSort: PropTypes.arrayOf(PropTypes.shape({})),
    sortHandler: PropTypes.func,
    entityType: PropTypes.string.isRequired,
};

TableWidget.defaultProps = {
    idAttribute: 'id',
    pageSize: DEFAULT_PAGE_SIZE,
    parentPageState: null,
    currentSort: null,
    sortHandler: null,
};
export default withRouter(TableWidget);
