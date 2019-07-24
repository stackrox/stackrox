import React from 'react';
import entityTypes from 'constants/entityTypes';
import URLService from 'modules/URLService';
import pluralize from 'pluralize';

import { sortValueByLength } from 'sorters/sorters';
import { NAMESPACES_QUERY as QUERY } from 'queries/namespace';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import queryService from 'modules/queryService';
import { CLIENT_SIDE_SEARCH_OPTIONS as SEARCH_OPTIONS } from 'constants/searchOptions';

import LabelChip from 'Components/LabelChip';
import List from './List';
import TableCellLink from './Link';

import filterByPolicyStatus from './utilities/filterByPolicyStatus';

const buildTableColumns = (match, location) => {
    const tableColumns = [
        {
            Header: 'Id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'metadata.id'
        },
        {
            Header: `Namespace`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'metadata.name'
        },
        {
            Header: `Cluster`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'metadata.clusterName',
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { metadata } = original;
                if (!metadata) return '-';
                const { clusterName, clusterId, id } = metadata;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.CLUSTER, clusterId)
                    .url();
                return <TableCellLink pdf={pdf} url={url} text={clusterName} />;
            }
        },
        {
            Header: `Policies Violated`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { policyStatus, metadata } = original;
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
                const { id } = metadata;
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
            Header: `Secrets`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { numSecrets, metadata } = original;
                if (!metadata || numSecrets === 0) return 'No Secrets';
                const { id } = metadata;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.SECRET)
                    .url();
                return (
                    <TableCellLink
                        pdf={pdf}
                        url={url}
                        text={`${numSecrets} ${pluralize('Secrets', numSecrets)}`}
                    />
                );
            },
            id: 'numSecrets',
            accessor: d => d.numSecrets,
            sortMethod: sortValueByLength
        },
        {
            Header: `Users & Groups`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { subjectsCount, metadata } = original;
                if (!subjectsCount || subjectsCount === 0) return 'No Users & Groups';
                const { id } = metadata;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.SUBJECT)
                    .url();
                return (
                    <TableCellLink
                        pdf={pdf}
                        url={url}
                        text={`${subjectsCount} ${pluralize('Users & Groups', subjectsCount)}`}
                    />
                );
            },
            accessor: 'subjectCount'
        },
        {
            Header: `Service Accounts`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { serviceAccountCount, metadata } = original;
                if (!serviceAccountCount || serviceAccountCount === 0) return 'No Service Accounts';
                const { id } = metadata;
                const url = URLService.getURL(match, location)
                    .push(id)
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
            accessor: 'serviceAccountCount'
        },
        {
            Header: `Roles granted`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { k8sroleCount, metadata } = original;
                if (!k8sroleCount || k8sroleCount === 0) return 'No Roles';
                const { id } = metadata;
                const url = URLService.getURL(match, location)
                    .push(id)
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
            accessor: 'k8sroleCount'
        }
    ];
    return tableColumns;
};

const createTableRows = data => data.results;

const Namespaces = ({ match, location, className, selectedRowId, onRowClick, query }) => {
    const tableColumns = buildTableColumns(match, location);
    const { [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus, ...restQuery } = query || {};
    const queryText = queryService.objectToWhereClause({ ...restQuery });
    const variables = queryText ? { query: queryText } : null;

    function createTableRowsFilteredByPolicyStatus(data) {
        const tableRows = createTableRows(data);
        const filteredTableRows = filterByPolicyStatus(tableRows, policyStatus);
        return filteredTableRows;
    }

    return (
        <List
            className={className}
            query={QUERY}
            variables={variables}
            entityType={entityTypes.NAMESPACE}
            tableColumns={tableColumns}
            createTableRows={createTableRowsFilteredByPolicyStatus}
            onRowClick={onRowClick}
            selectedRowId={selectedRowId}
            idAttribute="metadata.id"
            defaultSearchOptions={[SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]}
        />
    );
};
Namespaces.propTypes = entityListPropTypes;
Namespaces.defaultProps = entityListDefaultprops;

export default Namespaces;
