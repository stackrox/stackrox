import React from 'react';
import gql from 'graphql-tag';

import queryService from 'modules/queryService';
import TableCellLink from 'Components/TableCellLink';
import TableCountLink from 'Components/workflow/TableCountLink';
import CVEStackedPill from 'Components/CVEStackedPill';
import StatusChip from 'Components/StatusChip';
import DateTimeField from 'Components/DateTimeField';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import entityTypes from 'constants/entityTypes';
import { LIST_PAGE_SIZE } from 'constants/workflowPages.constants';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';
import { NAMESPACE_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';
import removeEntityContextColumns from 'utils/tableUtils';
import { namespaceSortFields } from 'constants/sortFields';

export const defaultNamespaceSort = [
    // @TODO, uncomment the primary sort field for Namespaces, after its available for backend pagination/sorting
    // {
    //     id: namespaceSortFields.PRIORITY,
    //     desc: false
    // },
    {
        id: namespaceSortFields.NAMESPACE,
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
            accessor: 'metadata.name',
            sortField: namespaceSortFields.NAMESPACE
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
            accessor: 'vulnCounter.all.total',
            sortField: namespaceSortFields.CVES
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
            accessor: 'metadata.clusterName',
            sortField: namespaceSortFields.CLUSTER
        },
        {
            Header: `Deployments`,
            entityType: entityTypes.DEPLOYMENT,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => (
                <TableCountLink
                    entityType={entityTypes.DEPLOYMENT}
                    count={original.deploymentCount}
                    textOnly={pdf}
                    selectedRowId={original.metadata.id}
                />
            ),
            accessor: 'deploymentCount',
            sortField: namespaceSortFields.DEPLOYMENTS
        },
        {
            Header: `Images`,
            entityType: entityTypes.IMAGE,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => (
                <TableCountLink
                    entityType={entityTypes.IMAGE}
                    count={original.imageCount}
                    textOnly={pdf}
                    selectedRowId={original.metadata.id}
                />
            ),
            accessor: 'imageCount',
            sortField: namespaceSortFields.IMAGES
        },
        {
            Header: `Policies`,
            entityType: entityTypes.POLICY,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => (
                <TableCountLink
                    entityType={entityTypes.POLICY}
                    count={original.policyCount}
                    textOnly={pdf}
                    selectedRowId={original.metadata.id}
                />
            ),
            accessor: 'policyCount',
            sortField: namespaceSortFields.POLICIES
        },
        {
            Header: `Policy Status`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original }) => {
                const { policyStatus } = original;
                const policyLabel = <StatusChip status={policyStatus && policyStatus.status} />;

                return policyLabel;
            },
            id: 'policyStatus',
            accessor: 'policyStatus.status',
            sortField: namespaceSortFields.POLICY_STATUS
        },
        {
            Header: `Latest Violation`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { latestViolation } = original;
                return <DateTimeField date={latestViolation} asString={pdf} />;
            },
            accessor: 'latestViolation',
            sortField: namespaceSortFields.LATEST_VIOLATION
        },
        {
            Header: `Risk Priority`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            accessor: 'metadata.priority',
            sortField: namespaceSortFields.PRIORITY
        }
    ];
    return removeEntityContextColumns(tableColumns, workflowState);
}

const VulnMgmtNamespaces = ({ selectedRowId, search, sort, page, data }) => {
    const query = gql`
        query getNamespaces($query: String, $policyQuery: String, $pagination: Pagination) {
            results: namespaces(query: $query, pagination: $pagination) {
                ...namespaceFields
            }
            count: namespaceCount(query: $query)
        }
        ${NAMESPACE_LIST_FRAGMENT}
    `;
    const tableSort = sort || defaultNamespaceSort;
    const queryOptions = {
        variables: {
            query: queryService.objectToWhereClause(search),
            policyQuery: queryService.objectToWhereClause({
                Category: 'Vulnerability Management'
            }),
            pagination: queryService.getPagination(tableSort, page, LIST_PAGE_SIZE)
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
            page={page}
        />
    );
};

VulnMgmtNamespaces.propTypes = workflowListPropTypes;
VulnMgmtNamespaces.defaultProps = {
    ...workflowListDefaultProps,
    sort: null
};

export default VulnMgmtNamespaces;
