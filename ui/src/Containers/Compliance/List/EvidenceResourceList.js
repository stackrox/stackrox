import React, { useState } from 'react';
import PropTypes from 'prop-types';
import Raven from 'raven-js';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import {
    COMPLIANCE_DATA_ON_NODES,
    COMPLIANCE_DATA_ON_DEPLOYMENTS,
    COMPLIANCE_DATA_ON_CLUSTERS
} from 'queries/table';
import URLService from 'modules/URLService';
import queryService from 'modules/queryService';
import uniq from 'lodash/uniq';
import upperCase from 'lodash/upperCase';
import pluralize from 'pluralize';
import createPDFTable from 'utils/pdfUtils';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import Panel from 'Components/Panel';
import Table from 'Components/Table';
import TablePagination from 'Components/TablePagination';
import entityTypes from 'constants/entityTypes';
import { resourceLabels } from 'messages/common';
import {
    nodesTableColumns,
    deploymentsTableColumns,
    clustersTableColumns
} from 'Containers/Compliance/List/evidenceTableColumns';

const getQueryVariables = params => {
    const { controlId, query } = params;
    const newQuery = {
        ...query,
        'Control ID': controlId
    };
    const variables = {
        query: queryService.objectToWhereClause(newQuery)
    };
    return variables;
};

const getQuery = resourceType => {
    let query;
    switch (resourceType) {
        case entityTypes.CLUSTER:
            query = COMPLIANCE_DATA_ON_CLUSTERS;
            break;
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
        case entityTypes.CLUSTER:
            tableColumn = clustersTableColumns;
            break;
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
    if (resourceType === entityTypes.CLUSTER) {
        data.results.forEach(cluster => {
            cluster.complianceResults.forEach(result => {
                // eslint-disable-next-line
                if (upperCase(result.resource.__typename) === resourceType) {
                    rows.push(result);
                }
            });
        });
    }
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

const processNumResources = (data, resourceType) => {
    try {
        let result = 0;
        if (resourceType === entityTypes.DEPLOYMENT) {
            result = data.results.reduce((acc, curr) => acc + curr.deployments.length, 0);
        }
        if (resourceType === entityTypes.NODE) {
            result = data.results.reduce((acc, curr) => acc + curr.nodes.length, 0);
        }
        if (resourceType === entityTypes.CLUSTER) {
            result = data.results.length;
        }
        return result;
    } catch (error) {
        Raven.captureException(error);
        return null;
    }
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
                const numResources = processNumResources(data, resourceType);
                const label =
                    numResources === 1
                        ? resourceLabels[resourceType]
                        : pluralize(resourceLabels[resourceType]);
                const header = `${numResources} ${label}`;
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
                createPDFTable(tableData, params.entityType, null, 'capture-list', resourceType);
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
