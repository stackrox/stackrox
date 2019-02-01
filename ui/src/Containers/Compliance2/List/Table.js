import React from 'react';
import PropTypes from 'prop-types';

import Table from 'Components/Table';
import Panel from 'Components/Panel';
import Loader from 'Components/Loader';

import TablePagination from 'Components/TablePagination';
import TableGroup from 'Components/TableGroup';
import entityToColumns from 'constants/tableColumns';
import { groupedData } from 'mockData/tableDataMock';
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
                contents =
                    params.entityType === 'compliance' ? (
                        <TableGroup
                            groups={groupedData}
                            tableColumns={entityToColumns[params.entityType]}
                            onRowClick={updateSelectedRow}
                            idAttribute={params.entityType}
                            selectedRowId={selectedRow ? selectedRow[params.entityType] : null}
                        />
                    ) : (
                        <Table
                            rows={tableData}
                            columns={entityToColumns[params.entityType]}
                            onRowClick={updateSelectedRow}
                            idAttribute="node"
                            selectedRowId={selectedRow ? selectedRow.node : null}
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
                <Panel header={params.entityType} headerComponents={paginationComponent}>
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
