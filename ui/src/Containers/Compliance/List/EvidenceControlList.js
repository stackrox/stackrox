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
import { standardLabels } from 'messages/standards';
import URLService from 'modules/URLService';
import queryService from 'modules/queryService';
import { sortValue } from 'sorters/sorters';
import uniq from 'lodash/uniq';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import Panel from 'Components/Panel';
import Table, { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import TablePagination from 'Components/TablePagination';
import ComplianceStateLabel from 'Containers/Compliance/ComplianceStateLabel';
import entityTypes from 'constants/entityTypes';

const tableColumns = [
    {
        Header: 'id',
        headerClassName: 'hidden',
        className: 'hidden',
        accessor: 'control.id'
    },
    {
        Header: `Standard`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'control.standardId',
        Cell: ({ original }) => standardLabels[original.control.standardId]
    },
    {
        Header: `Control`,
        headerClassName: `w-1/4 ${defaultHeaderClassName}`,
        className: `w-1/4 ${defaultColumnClassName}`,
        accessor: 'control.name',
        sortMethod: sortValue,
        Cell: ({ original }) => `${original.control.name} - ${original.control.description}`
    },
    {
        Header: `State`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'value.overallState',
        // eslint-disable-next-line
        Cell: ({ original }) => <ComplianceStateLabel state={original.value.overallState} />
    },
    {
        Header: `Entity`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'resource.name'
    },
    {
        Header: `Type`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'resource.__typename'
    },
    {
        Header: `Namespace`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'resource.namespace',
        Cell: ({ original }) => original.resource.namespace || '-'
    },
    {
        Header: `Evidence`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'value.evidence',
        // eslint-disable-next-line
        Cell: ({ original }) => {
            const { length } = original.value.evidence;
            return length > 1 ? (
                <div className="italic font-800">{`Inspect to view ${length} pieces of evidence`}</div>
            ) : (
                original.value.evidence[0].message
            );
        }
    }
];

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
