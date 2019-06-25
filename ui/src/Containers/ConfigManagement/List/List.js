import React, { useState } from 'react';
import PropTypes from 'prop-types';
import entityLabels from 'messages/entity';
import pluralize from 'pluralize';
import createPDFTable from 'utils/pdfUtils';
import resolvePath from 'object-resolve-path';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import Panel from 'Components/Panel';
import Table from 'Components/Table';
import TablePagination from 'Components/TablePagination';

const List = ({
    className,
    headerText,
    query,
    variables,
    entityType,
    tableColumns,
    createTableRows,
    onRowClick,
    selectedRowId,
    idAttribute
}) => {
    const [page, setPage] = useState(0);

    function onRowClickHandler(row) {
        const id = resolvePath(row, idAttribute);
        onRowClick(id);
    }

    return (
        <Query query={query} variables={variables}>
            {({ loading, data }) => {
                if (loading) return <Loader />;
                if (!data) return <PageNotFound resourceType={entityType} />;
                const tableRows = createTableRows(data);
                const header = `${tableRows.length} ${pluralize(
                    headerText || entityLabels[entityType]
                )}`;
                const headerComponents = (
                    <TablePagination page={page} dataLength={tableRows.length} setPage={setPage} />
                );
                if (tableRows.length) {
                    createPDFTable(tableRows, entityType, query, 'capture-list', tableColumns);
                }
                return (
                    <section id="capture-list" className="w-full">
                        <Panel
                            className={className}
                            header={header}
                            headerComponents={headerComponents}
                        >
                            <Table
                                rows={tableRows}
                                columns={tableColumns}
                                onRowClick={onRowClickHandler}
                                idAttribute={idAttribute}
                                id="capture-list"
                                selectedRowId={selectedRowId}
                                noDataText="No results found. Please refine your search."
                                page={page}
                            />
                        </Panel>
                    </section>
                );
            }}
        </Query>
    );
};

List.propTypes = {
    className: PropTypes.string,
    query: PropTypes.shape({}).isRequired,
    variables: PropTypes.shape({}),
    entityType: PropTypes.string.isRequired,
    tableColumns: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    createTableRows: PropTypes.func.isRequired,
    onRowClick: PropTypes.func.isRequired,
    selectedRowId: PropTypes.string,
    idAttribute: PropTypes.string.isRequired,
    headerText: PropTypes.string
};

List.defaultProps = {
    className: '',
    variables: {},
    headerText: '',
    selectedRowId: null
};

export default List;
