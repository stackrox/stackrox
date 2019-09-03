import React from 'react';
import gql from 'graphql-tag';
import entityTypes from 'constants/entityTypes';
import URLService from 'modules/URLService';

import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import { CLIENT_SIDE_SEARCH_OPTIONS as SEARCH_OPTIONS } from 'constants/searchOptions';

import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import LabelChip from 'Components/LabelChip';
import queryService from 'modules/queryService';
import pluralize from 'pluralize';
import List from './List';
import TableCellLink from './Link';

import filterByPolicyStatus from './utilities/filterByPolicyStatus';

const QUERY = gql`
    query clusters($query: String) {
        results: clusters(query: $query) {
            id
            name
            serviceAccountCount
            k8sroleCount
            subjectCount
            status {
                orchestratorMetadata {
                    version
                }
            }
            complianceControlCount(query: "Standard:CIS") {
                passingCount
                failingCount
                unknownCount
            }
            policyStatus {
                status
                failingPolicies {
                    id
                    name
                }
            }
        }
    }
`;

const buildTableColumns = (match, location) => {
    const tableColumns = [
        {
            Header: 'Id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'id'
        },
        {
            Header: `Cluster`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'name'
        },
        {
            Header: `K8S Version`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'status.orchestratorMetadata.version'
        },
        {
            Header: `Policy Status`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original }) => {
                const { policyStatus } = original;
                const { length } = policyStatus.failingPolicies;
                return !length ? 'Pass' : <LabelChip text="Fail" type="alert" />;
            },
            id: 'status',
            accessor: d => d.policyStatus.status
        },
        {
            Header: `CIS Controls`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'complianceControlCount',
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { complianceControlCount } = original;
                const { passingCount, failingCount, unknownCount } = complianceControlCount;
                const totalCount = passingCount + failingCount + unknownCount;
                if (!totalCount) {
                    return <LabelChip text="No Controls" type="alert" />;
                }
                const url = URLService.getURL(match, location)
                    .push(original.id)
                    .push(entityTypes.CONTROL)
                    .url();
                return (
                    <TableCellLink
                        pdf={pdf}
                        url={url}
                        text={`${totalCount} ${pluralize('Controls', totalCount)}`}
                    />
                );
            }
        },
        {
            Header: `Users & Groups`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { subjectCount } = original;
                if (!subjectCount) {
                    return <LabelChip text="No Users & Groups" type="alert" />;
                }
                const url = URLService.getURL(match, location)
                    .push(original.id)
                    .push(entityTypes.SUBJECT)
                    .url();
                return (
                    <TableCellLink
                        pdf={pdf}
                        url={url}
                        text={`${subjectCount} ${pluralize('Users & Groups', subjectCount)}`}
                    />
                );
            },
            id: 'subjectCount',
            accessor: d => d.subjectCount
        },
        {
            Header: `Service Accounts`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { serviceAccountCount } = original;
                if (!serviceAccountCount) {
                    return <LabelChip text="No Service Accounts" type="alert" />;
                }
                const url = URLService.getURL(match, location)
                    .push(original.id)
                    .push(entityTypes.SERVICE_ACCOUNT)
                    .url();
                return (
                    <TableCellLink
                        pdf={pdf}
                        url={url}
                        text={`${serviceAccountCount} ${pluralize(
                            'Service Accounts',
                            serviceAccountCount
                        )}`}
                    />
                );
            },
            id: 'serviceAccountCount',
            accessor: d => d.serviceAccountCount
        },
        {
            Header: `Roles`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { k8sroleCount } = original;
                if (!k8sroleCount) return <LabelChip text="No Roles" type="alert" />;
                const url = URLService.getURL(match, location)
                    .push(original.id)
                    .push(entityTypes.ROLE)
                    .url();
                return (
                    <TableCellLink
                        pdf={pdf}
                        url={url}
                        text={`${k8sroleCount} ${pluralize('Roles', k8sroleCount)}`}
                    />
                );
            },
            id: 'k8sroleCount',
            accessor: d => d.k8sroleCount
        }
    ];
    return tableColumns;
};

const createTableRows = data => data.results;

const Clusters = ({ match, location, className, selectedRowId, onRowClick, query, data }) => {
    const tableColumns = buildTableColumns(match, location);
    const { [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus, ...restQuery } = query || {};
    const queryObject = { ...restQuery };
    const queryText = queryService.objectToWhereClause(queryObject);
    const variables = queryText ? { query: queryText } : null;

    function createTableRowsFilteredByPolicyStatus(items) {
        const tableRows = createTableRows(items);
        const filteredTableRows = filterByPolicyStatus(tableRows, policyStatus);
        return filteredTableRows;
    }

    return (
        <List
            className={className}
            query={QUERY}
            variables={variables}
            entityType={entityTypes.CLUSTER}
            tableColumns={tableColumns}
            createTableRows={createTableRowsFilteredByPolicyStatus}
            onRowClick={onRowClick}
            selectedRowId={selectedRowId}
            idAttribute="id"
            defaultSearchOptions={[SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]}
            data={filterByPolicyStatus(data, policyStatus)}
        />
    );
};

Clusters.propTypes = entityListPropTypes;
Clusters.defaultProps = entityListDefaultprops;

export default Clusters;
