import React from 'react';
import entityTypes from 'constants/entityTypes';
import { DEPLOYMENTS_QUERY as QUERY } from 'queries/deployment';
import URLService from 'modules/URLService';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import { sortValueByLength } from 'sorters/sorters';
import { CLIENT_SIDE_SEARCH_OPTIONS as SEARCH_OPTIONS } from 'constants/searchOptions';

import queryService from 'modules/queryService';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import LabelChip from 'Components/LabelChip';
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
            Header: `Deployment`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'name'
        },
        {
            Header: `Cluster`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'clusterName',
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { clusterName, clusterId, id } = original;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.CLUSTER, clusterId)
                    .url();
                return <TableCellLink pdf={pdf} url={url} text={clusterName} />;
            }
        },
        {
            Header: `Namespace`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'namespace',
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { namespace, namespaceId, id } = original;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.NAMESPACE, namespaceId)
                    .url();
                return <TableCellLink pdf={pdf} url={url} text={namespace} />;
            }
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
            Header: `Images`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { imagesCount, id } = original;
                if (imagesCount === 0) return 'No images';
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.IMAGE)
                    .url();
                return <TableCellLink pdf={pdf} url={url} text={`${imagesCount} image(s)`} />;
            },
            accessor: 'imagesCount'
        },
        {
            Header: `Secrets`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { secretCount, id } = original;
                if (secretCount === 0) return 'No secrets';
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.SECRET)
                    .url();
                return <TableCellLink pdf={pdf} url={url} text={`${secretCount} secret(s)`} />;
            },
            accessor: 'secretCount'
        },
        {
            Header: `Service Account`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'serviceAccount',
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { serviceAccount, id } = original;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.SERVICE_ACCOUNT)
                    .url();
                return <TableCellLink pdf={pdf} url={url} text={serviceAccount} />;
            }
        }
    ];
    return tableColumns;
};

const createTableRows = data => data.results;

const Deployments = ({ match, location, className, selectedRowId, onRowClick, query, data }) => {
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
            entityType={entityTypes.DEPLOYMENT}
            tableColumns={tableColumns}
            createTableRows={createTableRowsFilteredByPolicyStatus}
            onRowClick={onRowClick}
            selectedRowId={selectedRowId}
            idAttribute="id"
            defaultSorted={[
                {
                    id: 'deployAlertsCount',
                    desc: true
                }
            ]}
            defaultSearchOptions={[SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]}
            data={filterByPolicyStatus(data, policyStatus)}
        />
    );
};
Deployments.propTypes = entityListPropTypes;
Deployments.defaultProps = entityListDefaultprops;

export default Deployments;
