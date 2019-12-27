import React from 'react';
import gql from 'graphql-tag';

import queryService from 'modules/queryService';
import DateTimeField from 'Components/DateTimeField';
import StatusChip from 'Components/StatusChip';
import CVEStackedPill from 'Components/CVEStackedPill';
import TableCellLink from 'Components/TableCellLink';
import TableCountLink from 'Components/workflow/TableCountLink';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import entityTypes from 'constants/entityTypes';
import { LIST_PAGE_SIZE } from 'constants/workflowPages.constants';
import { DEPLOYMENT_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';
import removeEntityContextColumns from 'utils/tableUtils';
import { deploymentSortFields } from 'constants/sortFields';

export const defaultDeploymentSort = [
    {
        id: deploymentSortFields.PRIORITY,
        desc: false
    },
    {
        id: deploymentSortFields.DEPLOYMENT,
        desc: false
    }
];

export function getDeploymentTableColumns(workflowState) {
    // to determine whether to show the counts as links in the table when not in pure DEPLOYMENT state
    const inFindingsSection =
        workflowState.getCurrentEntity().entityType !== entityTypes.DEPLOYMENT;
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
            accessor: 'name',
            sortField: deploymentSortFields.DEPLOYMENT
        },
        {
            Header: `CVEs`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            entityType: entityTypes.CVE,
            Cell: ({ original, pdf }) => {
                const { vulnCounter, id } = original;
                if (!vulnCounter || (vulnCounter.all && vulnCounter.all.total === 0))
                    return 'No CVEs';

                const newState = workflowState.pushListItem(id).pushList(entityTypes.CVE);
                const url = newState.toUrl();

                // If `Fixed By` is set, it means vulnerability is fixable.
                const fixableUrl = newState.setSearch({ 'Fixed By': 'r/.*' }).toUrl();

                return (
                    <CVEStackedPill
                        vulnCounter={vulnCounter}
                        url={url}
                        fixableUrl={fixableUrl}
                        hideLink={pdf}
                    />
                );
            },
            accessor: 'vulnCounter.all.total',
            sortField: deploymentSortFields.CVE_COUNT
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
            sortField: deploymentSortFields.LATEST_VIOLATION
        },
        {
            Header: `Policies`,
            entityType: entityTypes.POLICY,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            accessor: 'failingPolicyCount',
            Cell: ({ original, pdf }) => (
                <TableCountLink
                    entityType={entityTypes.POLICY}
                    count={original.failingPolicyCount}
                    textOnly={inFindingsSection || pdf}
                    selectedRowId={original.id}
                    entityTypeText="failing policy"
                />
            ),
            sortField: deploymentSortFields.POLICIES
        },
        {
            Header: `Policy Status`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { policyStatus } = original;
                const policyLabel = <StatusChip status={policyStatus} />;

                return policyLabel;
            },
            id: 'policyStatus',
            accessor: 'policyStatus',
            sortField: deploymentSortFields.POLICY_STATUS
        },
        {
            Header: `Cluster`,
            entityType: entityTypes.CLUSTER,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            accessor: 'clusterName',
            Cell: ({ original, pdf }) => {
                const { clusterName, clusterId, id } = original;
                const url = workflowState
                    .pushListItem(id)
                    .pushRelatedEntity(entityTypes.CLUSTER, clusterId)
                    .toUrl();
                return (
                    <TableCellLink pdf={inFindingsSection || pdf} url={url} text={clusterName} />
                );
            },
            sortField: deploymentSortFields.CLUSTER
        },
        {
            Header: `Namespace`,
            entityType: entityTypes.NAMESPACE,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'namespace',
            Cell: ({ original, pdf }) => {
                const { namespace, namespaceId, id } = original;
                const url = workflowState
                    .pushListItem(id)
                    .pushRelatedEntity(entityTypes.NAMESPACE, namespaceId)
                    .toUrl();
                return <TableCellLink pdf={inFindingsSection || pdf} url={url} text={namespace} />;
            },
            sortField: deploymentSortFields.NAMESPACE
        },
        {
            Header: `Images`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => (
                <TableCountLink
                    entityType={entityTypes.IMAGE}
                    count={original.imageCount}
                    textOnly={inFindingsSection || pdf}
                    selectedRowId={original.id}
                />
            ),
            accessor: 'imageCount',
            sortField: deploymentSortFields.IMAGES
        },
        {
            Header: `Risk Priority`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            accessor: 'priority',
            sortField: deploymentSortFields.PRIORITY
        }
    ];
    return removeEntityContextColumns(tableColumns, workflowState);
}

const VulnMgmtDeployments = ({ selectedRowId, search, sort, page, data }) => {
    const query = gql`
        query getDeployments($query: String, $policyQuery: String, $pagination: Pagination) {
            results: deployments(query: $query, pagination: $pagination) {
                ...deploymentFields
            }
            count: deploymentCount(query: $query)
        }
        ${DEPLOYMENT_LIST_FRAGMENT}
    `;
    const tableSort = sort || defaultDeploymentSort;
    const queryOptions = {
        variables: {
            policyQuery: queryService.objectToWhereClause({
                Category: 'Vulnerability Management'
            }),
            query: queryService.objectToWhereClause(search),
            pagination: queryService.getPagination(tableSort, page, LIST_PAGE_SIZE)
        }
    };

    return (
        <WorkflowListPage
            data={data}
            query={query}
            queryOptions={queryOptions}
            idAttribute="id"
            entityListType={entityTypes.DEPLOYMENT}
            getTableColumns={getDeploymentTableColumns}
            selectedRowId={selectedRowId}
            search={search}
            page={page}
        />
    );
};

VulnMgmtDeployments.propTypes = workflowListPropTypes;
VulnMgmtDeployments.defaultProps = {
    ...workflowListDefaultProps,
    sort: null
};

export default VulnMgmtDeployments;
