import React from 'react';
import gql from 'graphql-tag';
import pluralize from 'pluralize';

import queryService from 'modules/queryService';
import TableCellLink from 'Components/TableCellLink';
import CVEStackedPill from 'Components/CVEStackedPill';
import StatusChip from 'Components/StatusChip';
import DateTimeField from 'Components/DateTimeField';
import { sortDate } from 'sorters/sorters';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import entityTypes from 'constants/entityTypes';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';
import { NAMESPACE_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';
import removeEntityContextColumns from 'utils/tableUtils';

export const defaultNamespaceSort = [
    {
        id: 'metadata.priority',
        desc: false
    },
    {
        id: 'metadata.name',
        desc: false
    }
];

export function getNamespaceTableColumns(workflowState) {
    const tableColumns = [
        {
            Header: 'Id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'metadata.id'
        },
        {
            Header: `Namespace`,
            headerClassName: `w-1/6 ${defaultHeaderClassName}`,
            className: `w-1/6 ${defaultColumnClassName}`,
            accessor: 'metadata.name'
        },
        {
            Header: `CVEs`,
            entityType: entityTypes.CVE,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { vulnCounter, metadata } = original;
                if (!vulnCounter || vulnCounter.all.total === 0) return 'No CVEs';

                const newState = workflowState.pushListItem(metadata.id).pushList(entityTypes.CVE);
                const cvesUrl = newState.toUrl();

                // If `Fixed By` is set, it means vulnerability is fixable.
                const fixableUrl = newState.setSearch({ 'Fixed By': 'r/.*' }).toUrl();

                return (
                    <CVEStackedPill
                        vulnCounter={vulnCounter}
                        url={cvesUrl}
                        fixableUrl={fixableUrl}
                        hideLink={pdf}
                    />
                );
            },
            accessor: 'vulnCounter.all.total'
        },
        {
            Header: `Cluster`,
            entityType: entityTypes.CLUSTER,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { metadata } = original;
                const { clusterName, clusterId, id } = metadata;
                const url = workflowState
                    .pushListItem(id)
                    .pushRelatedEntity(entityTypes.CLUSTER, clusterId)
                    .toUrl();

                return <TableCellLink pdf={pdf} url={url} text={clusterName} />;
            },
            accessor: 'metadata.clusterName'
        },
        {
            Header: `Deployments`,
            entityType: entityTypes.DEPLOYMENT,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { deploymentCount, metadata } = original;
                const url = workflowState
                    .pushListItem(metadata.id)
                    .pushList(entityTypes.DEPLOYMENT)
                    .toUrl();

                const text = `${deploymentCount} ${pluralize(
                    entityTypes.DEPLOYMENT.toLowerCase(),
                    deploymentCount
                )}`;
                return <TableCellLink pdf={pdf} url={url} text={text} />;
            },
            accessor: 'deploymentCount'
        },
        {
            Header: `Images`,
            entityType: entityTypes.IMAGE,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { imageCount, metadata } = original;
                const url = workflowState
                    .pushListItem(metadata.id)
                    .pushList(entityTypes.IMAGE)
                    .toUrl();

                const text = `${imageCount} ${pluralize(
                    entityTypes.IMAGE.toLowerCase(),
                    imageCount
                )}`;
                return <TableCellLink pdf={pdf} url={url} text={text} />;
            },
            accessor: 'imageCount'
        },
        {
            Header: `Policies`,
            entityType: entityTypes.POLICY,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { policyCount, metadata } = original;
                const url = workflowState
                    .pushListItem(metadata.id)
                    .pushList(entityTypes.POLICY)
                    .toUrl();
                const text = `${policyCount} ${pluralize(
                    entityTypes.POLICY.toLowerCase(),
                    policyCount
                )}`;
                return <TableCellLink pdf={pdf} url={url} text={text} />;
            },
            accessor: 'policyCount'
        },
        {
            Header: `Policy status`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original }) => {
                const { policyStatus } = original;
                const policyLabel = <StatusChip status={policyStatus && policyStatus.status} />;

                return policyLabel;
            },
            id: 'policyStatus',
            accessor: 'policyStatus.status'
        },
        {
            Header: `Latest violation`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { latestViolation } = original;
                return <DateTimeField date={latestViolation} />;
            },
            accessor: 'latestViolation',
            sortMethod: sortDate
        },
        {
            Header: `Risk Priority`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            accessor: 'metadata.priority'
        }
    ];
    return removeEntityContextColumns(tableColumns, workflowState);
}

const VulnMgmtNamespaces = ({ selectedRowId, search, sort, page, data }) => {
    const query = gql`
        query getNamespaces($query: String, $policyQuery: String) {
            results: namespaces(query: $query) {
                ...namespaceFields
            }
        }
        ${NAMESPACE_LIST_FRAGMENT}
    `;

    const queryOptions = {
        variables: {
            query: queryService.objectToWhereClause(search),
            policyQuery: queryService.objectToWhereClause({
                Category: 'Vulnerability Management'
            })
        }
    };

    return (
        <WorkflowListPage
            data={data}
            query={query}
            queryOptions={queryOptions}
            entityListType={entityTypes.NAMESPACE}
            getTableColumns={getNamespaceTableColumns}
            selectedRowId={selectedRowId}
            idAttribute="metadata.id"
            search={search}
            sort={sort}
            page={page}
            defaultSorted={sort}
        />
    );
};

VulnMgmtNamespaces.propTypes = workflowListPropTypes;
VulnMgmtNamespaces.defaultProps = {
    ...workflowListDefaultProps,
    sort: defaultNamespaceSort
};

export default VulnMgmtNamespaces;
