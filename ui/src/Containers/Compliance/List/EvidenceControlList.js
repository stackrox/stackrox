import React, { useState } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import {
    COMPLIANCE_DATA_ON_CLUSTERS,
    COMPLIANCE_DATA_ON_CLUSTER,
    COMPLIANCE_DATA_ON_NAMESPACE,
    COMPLIANCE_DATA_ON_NODE,
    COMPLIANCE_DATA_ON_DEPLOYMENT
} from 'queries/table';
import URLService from 'modules/URLService';
import queryService from 'modules/queryService';
import uniq from 'lodash/uniq';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import Panel from 'Components/Panel';
import Table from 'Components/Table';
import TablePagination from 'Components/TablePagination';
import entityTypes from 'constants/entityTypes';

import { controlsTableColumns as tableColumns } from 'Containers/Compliance/List/evidenceTableColumns';

const getQueryVariables = params => {
    const { entityType, entityId, query } = params;
    const variables = {
        id: entityType === entityTypes.CONTROL ? null : entityId,
        where: queryService.objectToWhereClause(query)
    };
    return variables;
};

const getQuery = params => {
    const { entityType } = params;
    let query;
    switch (entityType) {
        case entityTypes.CLUSTER:
            query = COMPLIANCE_DATA_ON_CLUSTER;
            break;
        case entityTypes.NAMESPACE:
            query = COMPLIANCE_DATA_ON_NAMESPACE;
            break;
        case entityTypes.NODE:
            query = COMPLIANCE_DATA_ON_NODE;
            break;
        case entityTypes.DEPLOYMENT:
            query = COMPLIANCE_DATA_ON_DEPLOYMENT;
            break;
        case entityTypes.CONTROL:
            query = COMPLIANCE_DATA_ON_CLUSTERS;
            break;
        default:
            break;
    }
    return query;
};

const createTableRows = data => {
    if (!data || (!data.results && !data.result)) return [];
    let rows = [];
    if (data.results) {
        data.results.forEach(resource => {
            const { complianceResults } = resource;
            rows = [...rows, ...complianceResults];
        });
    } else if (data.result) {
        rows = data.result.complianceResults;
    }
    return rows;
};

const createTableData = data => {
    const rows = createTableRows(data);
    const columns = tableColumns;
    const numControls = uniq(rows.map(row => row.control.id)).length;
    return {
        numControls,
        rows,
        columns
    };
};

const EvidenceControlList = ({ selectedRow, updateSelectedRow, match, location }) => {
    const [page, setPage] = useState(0);
    const params = URLService.getParams(match, location);
    const variables = getQueryVariables(params);
    const query = getQuery(params);
    return (
        <Query query={query} variables={variables}>
            {({ loading, data }) => {
                if (loading) return <Loader />;
                const tableData = createTableData(data);
                const header = `${tableData.numControls} Controls`;
                return (
                    <Panel
                        header={header}
                        headerComponents={
                            <TablePagination
                                page={page}
                                dataLength={tableData.rows.length}
                                setPage={setPage}
                            />
                        }
                    >
                        <Table
                            rows={tableData.rows}
                            columns={tableData.columns}
                            onRowClick={updateSelectedRow}
                            idAttribute="control.id"
                            selectedRowId={selectedRow ? selectedRow.control.id : null}
                            noDataText="No results found. Please refine your search."
                            page={page}
                        />
                    </Panel>
                );
            }}
        </Query>
    );
};

EvidenceControlList.propTypes = {
    selectedRow: PropTypes.shape({}),
    updateSelectedRow: PropTypes.func.isRequired,
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired
};

EvidenceControlList.defaultProps = {
    selectedRow: null
};

export default withRouter(EvidenceControlList);
