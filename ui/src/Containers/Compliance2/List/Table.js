import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { standardTypes } from 'constants/entityTypes';
import standardLabels from 'messages/standards';

import Table from 'Components/Table';
import Panel from 'Components/Panel';
import Loader from 'Components/Loader';

import TablePagination from 'Components/TablePagination';
import TableGroup from 'Components/TableGroup';
import entityToColumns from 'constants/tableColumns';
import componentTypes from 'constants/componentTypes';
import AppQuery from 'Components/AppQuery';

const standardTypeValues = Object.values(standardTypes);

class ListTable extends Component {
    static propTypes = {
        params: PropTypes.shape({}).isRequired,
        selectedRow: PropTypes.shape({}),
        updateSelectedRow: PropTypes.func.isRequired
    };

    static defaultProps = {
        selectedRow: null
    };

    constructor(props) {
        super(props);
        this.state = {
            page: 0
        };
    }

    setTablePage = page => this.setState({ page });

    render() {
        const { params, selectedRow, updateSelectedRow } = this.props;
        const { page } = this.state;
        return (
            <AppQuery params={params} componentType={componentTypes.LIST_TABLE}>
                {({ loading, data }) => {
                    const isStandard = standardTypeValues.includes(params.entityType);
                    let tableData;
                    let contents = <Loader />;
                    let paginationComponent;
                    const headerText = isStandard
                        ? `${standardLabels[params.entityType]} controls`
                        : `${params.entityType}`;

                    if (!loading && data) {
                        tableData = data.results;
                        contents = isStandard ? (
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
                                setPage={this.setTablePage}
                            />
                        );
                    }
                    return (
                        <Panel header={headerText} headerComponents={paginationComponent}>
                            {contents}
                        </Panel>
                    );
                }}
            </AppQuery>
        );
    }
}

export default ListTable;
