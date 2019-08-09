import React from 'react';
import entityTypes from 'constants/entityTypes';
import URLService from 'modules/URLService';
import { sortValueByLength } from 'sorters/sorters';
import { CLUSTERS_QUERY as QUERY } from 'queries/cluster';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import { CLIENT_SIDE_SEARCH_OPTIONS as SEARCH_OPTIONS } from 'constants/searchOptions';

import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import LabelChip from 'Components/LabelChip';
import queryService from 'modules/queryService';
import pluralize from 'pluralize';
import List from './List';
import TableCellLink from './Link';

import filterByPolicyStatus from './utilities/filterByPolicyStatus';

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
            Header: `Policies Violated`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { policyStatus, id } = original;
                const { failingPolicies } = policyStatus;
                if (failingPolicies.length)
                    return (
                        <LabelChip
                            text={`${failingPolicies.length} ${pluralize(
                                'Policies',
                                failingPolicies.length
                            )}`}
                            type="alert"
                        />
                    );
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.POLICY)
                    .url();
                return <TableCellLink pdf={pdf} url={url} text="View Policies" />;
            },
            id: 'failingPolicies',
            accessor: d => d.policyStatus.failingPolicies,
            sortMethod: sortValueByLength
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
            accessor: 'complianceResults',
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { complianceResults = [] } = original;
                const filteredComplianceResults = complianceResults.filter(
                    // eslint-disable-next-line
                    result => result.resource.__typename === 'Cluster'
                );
                const { length } = filteredComplianceResults;
                if (!length) {
                    return <LabelChip text="No Controls" type="alert" />;
                }
                const url = URLService.getURL(match, location)
                    .push(original.id)
                    .push(entityTypes.CONTROL)
                    .url();
                if (length > 1)
                    return (
                        <TableCellLink
                            pdf={pdf}
                            url={url}
                            text={`${length} ${pluralize('Controls', length)}`}
                        />
                    );
                return original.complianceResults[0].control.name;
            }
        },
        {
            Header: `Users & Groups`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { length } = original.subjects;
                if (!length) {
                    return <LabelChip text="No Users & Groups" type="alert" />;
                }
                const url = URLService.getURL(match, location)
                    .push(original.id)
                    .push(entityTypes.SUBJECT)
                    .url();
                if (length > 1)
                    return (
                        <TableCellLink
                            pdf={pdf}
                            url={url}
                            text={`${length} ${pluralize('Users & Groups', length)}`}
                        />
                    );
                return original.subjects[0].id;
            },
            id: 'subjects',
            accessor: d => d.subjects,
            sortMethod: sortValueByLength
        },
        {
            Header: `Service Accounts`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { length } = original.serviceAccounts;
                if (!length) {
                    return <LabelChip text="No Service Accounts" type="alert" />;
                }
                const url = URLService.getURL(match, location)
                    .push(original.id)
                    .push(entityTypes.SERVICE_ACCOUNT)
                    .url();
                if (length > 1)
                    return (
                        <TableCellLink
                            pdf={pdf}
                            url={url}
                            text={`${length} ${pluralize('Service Accounts', length)}`}
                        />
                    );
                return original.serviceAccounts[0].name;
            },
            id: 'serviceAccounts',
            accessor: d => d.serviceAccounts,
            sortMethod: sortValueByLength
        },
        {
            Header: `Roles`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { length } = original.k8sroles;
                if (!length) return <LabelChip text="No Roles" type="alert" />;
                const url = URLService.getURL(match, location)
                    .push(original.id)
                    .push(entityTypes.ROLE)
                    .url();
                if (length > 1)
                    return (
                        <TableCellLink
                            pdf={pdf}
                            url={url}
                            text={`${length} ${pluralize('Roles', length)}`}
                        />
                    );
                return original.k8sroles[0].name;
            },
            id: 'k8sroles',
            accessor: d => d.k8sroles,
            sortMethod: sortValueByLength
        }
    ];
    return tableColumns;
};

const createTableRows = data => data.results;

const Clusters = ({ match, location, className, selectedRowId, onRowClick, query, data }) => {
    const tableColumns = buildTableColumns(match, location);
    const { [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus, ...restQuery } = query || {};
    const queryText = queryService.objectToWhereClause({ ...restQuery });
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
