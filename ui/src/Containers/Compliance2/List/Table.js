import React from 'react';
import PropTypes from 'prop-types';
import { standardTypes } from 'constants/entityTypes';

import Table from 'Components/Table';
import Panel from 'Components/Panel';
import Loader from 'Components/Loader';

import TablePagination from 'Components/TablePagination';
import TableGroup from 'Components/TableGroup';
import entityToColumns from 'constants/tableColumns';
import componentTypes from 'constants/componentTypes';
import AppQuery from 'Components/AppQuery';

const ListTable = ({ params, selectedRow, page, updateSelectedRow, setTablePage }) => (
    <AppQuery params={params} componentType={componentTypes.LIST_TABLE}>
        {({ loading, data }) => {
            let tableData;
            let contents = <Loader />;
            let paginationComponent;

            if (!loading && data) {
                tableData = data.results;
                contents = Object.values(standardTypes).includes(params.entityType) ? (
                    <TableGroup
                        groups={tableData}
                        tableColumns={entityToColumns[params.entityType]}
                        onRowClick={updateSelectedRow}
                        idAttribute="control"
                        selectedRowId={selectedRow ? selectedRow.control : null}
                    />
                ) : (
                    <Table
                        rows={tableData}
                        columns={entityToColumns[params.entityType]}
                        onRowClick={updateSelectedRow}
                        idAttribute="id"
                        selectedRowId={selectedRow ? selectedRow.id : null}
                        noDataText="No results found. Please refine your search."
                        page={page}
                    />
                );
                paginationComponent = (
                    <TablePagination
                        page={page}
                        dataLength={tableData.length}
                        setPage={setTablePage}
                    />
                );
            }
            return (
                <Panel
                    header={`${params.entityType} controls`}
                    headerComponents={paginationComponent}
                >
                    {contents}
                </Panel>
            );
        }}
    </AppQuery>
);

ListTable.propTypes = {
    params: PropTypes.shape({}).isRequired,
    selectedRow: PropTypes.shape({}),
    page: PropTypes.number.isRequired,
    updateSelectedRow: PropTypes.func.isRequired,
    setTablePage: PropTypes.func.isRequired
};

ListTable.defaultProps = {
    selectedRow: null
};

export default ListTable;
