import React, { useState } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import { COMPLIANCE_DATA_ON_NODES, COMPLIANCE_DATA_ON_DEPLOYMENTS } from 'queries/table';
import URLService from 'modules/URLService';
import queryService from 'modules/queryService';
import uniq from 'lodash/uniq';
import upperCase from 'lodash/upperCase';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import Panel from 'Components/Panel';
import Table from 'Components/Table';
import TablePagination from 'Components/TablePagination';
import entityTypes from 'constants/entityTypes';
import {
    nodesTableColumns,
    deploymentsTableColumns
} from 'Containers/Compliance/List/evidenceTableColumns';

const getQueryVariables = params => {
    const { controlId, query } = params;
    const newQuery = {
        ...query,
        'Control Id': controlId
    };
    const variables = {
        where: queryService.objectToWhereClause(newQuery)
    };
    return variables;
};

const getQuery = resourceType => {
    let query;
    switch (resourceType) {
        case entityTypes.DEPLOYMENT:
            query = COMPLIANCE_DATA_ON_DEPLOYMENTS;
            break;
        default:
            query = COMPLIANCE_DATA_ON_NODES;
            break;
    }
    return query;
};

const createTableColumn = resourceType => {
    let tableColumn;
    switch (resourceType) {
        case entityTypes.DEPLOYMENT:
            tableColumn = deploymentsTableColumns;
            break;
        default:
            tableColumn = nodesTableColumns;
            break;
    }
    return tableColumn;
};

const createTableRows = (data, resourceType) => {
    if (!data || !data.results) return [];
    let rows = [];
    if (resourceType === entityTypes.DEPLOYMENT) {
        data.results.forEach(cluster => {
            cluster.deployments.forEach(deployment => {
                deployment.complianceResults.forEach(result => {
                    // eslint-disable-next-line
                    if (upperCase(result.resource.__typename) === resourceType) {
                        rows.push(result);
                    }
                });
            });
        });
    }
    if (resourceType === entityTypes.NODE) {
        data.results.forEach(cluster => {
            cluster.nodes.forEach(node => {
                rows = [...rows, ...node.complianceResults];
            });
        });
    }
    return rows;
};

const createTableData = (data, resourceType) => {
    const rows = createTableRows(data, resourceType);
    const columns = createTableColumn(resourceType);
    const numControls = uniq(rows.map(row => row.control.id)).length;
    return {
        numControls,
        rows,
        columns
    };
};

const EvidenceResourceList = ({
    searchComponent,
    resourceType,
    selectedRow,
    updateSelectedRow,
    match,
    location
}) => {
    const [page, setPage] = useState(0);
    const params = URLService.getParams(match, location);
    const variables = getQueryVariables(params);
    const query = getQuery(resourceType);
    return (
        <Query query={query} variables={variables}>
            {({ loading, data }) => {
                if (loading) return <Loader />;
                const tableData = createTableData(data, resourceType);
                const header = `${tableData.numControls} Controls`;
                const headerComponents = (
                    <>
                        <div className="flex flex-1 justify-start">{searchComponent}</div>
                        <TablePagination
                            page={page}
                            dataLength={tableData.rows.length}
                            setPage={setPage}
                        />
                    </>
                );
                return (
                    <Panel header={header} headerComponents={headerComponents}>
                        <Table
                            rows={tableData.rows}
                            columns={tableData.columns}
                            onRowClick={updateSelectedRow}
                            idAttribute="control.id"
                            selectedRowId={
                                selectedRow && selectedRow.control ? selectedRow.control.id : null
                            }
                            noDataText="No results found. Please refine your search."
                            page={page}
                        />
                    </Panel>
                );
            }}
        </Query>
    );
};

EvidenceResourceList.propTypes = {
    searchComponent: PropTypes.node,
    resourceType: PropTypes.string.isRequired,
    selectedRow: PropTypes.shape({}),
    updateSelectedRow: PropTypes.func.isRequired,
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired
};

EvidenceResourceList.defaultProps = {
    searchComponent: null,
    selectedRow: null
};

export default withRouter(EvidenceResourceList);
