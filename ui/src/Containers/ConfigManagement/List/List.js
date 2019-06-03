import React, { useState } from 'react';
import PropTypes from 'prop-types';
import entityLabels from 'messages/entity';
import pluralize from 'pluralize';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import Panel from 'Components/Panel';
import Table from 'Components/Table';
import TablePagination from 'Components/TablePagination';

const List = ({ query, entityType, tableColumns, createTableRows, onRowClick }) => {
    const [selectedRow, setSelectedRow] = useState(null);
    const [page, setPage] = useState(0);

    function onRowClickHandler(row) {
        if (row === selectedRow) {
            setSelectedRow(null);
        } else {
            onRowClick(row.id);
            setSelectedRow(row);
        }
    }

    return (
        <Query query={query}>
            {({ loading, data }) => {
                if (loading) return <Loader />;
                if (!data) return <PageNotFound resourceType={entityType} />;
                const tableRows = createTableRows(data);
                const header = `${tableRows.length} ${pluralize(entityLabels[entityType])}`;
                const headerComponents = (
                    <TablePagination page={page} dataLength={tableRows.length} setPage={setPage} />
                );
                return (
                    <Panel header={header} headerComponents={headerComponents}>
                        <Table
                            rows={tableRows}
                            columns={tableColumns}
                            onRowClick={onRowClickHandler}
                            idAttribute="id"
                            selectedRowId={selectedRow ? selectedRow.id : null}
                            noDataText="No results found. Please refine your search."
                            page={page}
                        />
                    </Panel>
                );
            }}
        </Query>
    );
};

List.propTypes = {
    query: PropTypes.shape().isRequired,
    entityType: PropTypes.string.isRequired,
    tableColumns: PropTypes.arrayOf(PropTypes.shape).isRequired,
    createTableRows: PropTypes.func.isRequired,
    onRowClick: PropTypes.func.isRequired
};

export default List;
