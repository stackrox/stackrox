import React from 'react';
import gql from 'graphql-tag';

import queryService from 'modules/queryService';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import DateTimeField from 'Components/DateTimeField';
import StatusChip from 'Components/StatusChip';
import TableCountLink from 'Components/workflow/TableCountLink';
import entityTypes from 'constants/entityTypes';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';
import CVEStackedPill from 'Components/CVEStackedPill';

import { CLUSTER_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';
import { clusterSortFields } from 'constants/sortFields';
import { LIST_PAGE_SIZE } from 'constants/workflowPages.constants';
import removeEntityContextColumns from 'utils/tableUtils';

export const defaultClusterSort = [
    // @TODO, uncomment the primary sort field for Clusters, after its available for backend pagination/sorting
    // {
    //     id: clusterSortFields.PRIORITY,
    //     desc: false
    // },
    {
        id: clusterSortFields.CLUSTER,
        desc: false
    }
];

// @TODO: remove this exception, once Clusters pagination is fixed on the back end
// eslint-disable-next-line
const VulnMgmtClusters = ({ selectedRowId, search, sort, page, data }) => {
    const query = gql`
        query getClusters($query: String, $policyQuery: String, $pagination: Pagination) {
            results: clusters(query: $query, pagination: $pagination) {
                ...clusterFields
            }
            count: clusterCount(query: $query)
        }
        ${CLUSTER_LIST_FRAGMENT}
    `;

    // @TODO: uncomment once Clusters pagination is fixed on the back end
    // const tableSort = sort || defaultClusterSort;
    const queryOptions = {
        variables: {
            policyQuery: queryService.objectToWhereClause({
                Category: 'Vulnerability Management'
            }),
            query: queryService.objectToWhereClause(search),
            // @TODO: delete the following line
            //        and uncomment the line after that, once Clusters pagination is fixed on the back end
            pagination: queryService.getPagination({}, page, LIST_PAGE_SIZE)
            // pagination: queryService.getPagination(tableSort, page, LIST_PAGE_SIZE)
        }
    };

    function getTableColumns(workflowState) {
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
                accessor: 'name',
                sortField: clusterSortFields.CLUSTER
            },
            {
                Header: `CVEs`,
                entityType: entityTypes.CVE,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => {
                    const { vulnCounter, id } = original;
                    if (!vulnCounter || vulnCounter.all.total === 0) return 'No CVEs';

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
                sortField: clusterSortFields.CLUSTER
            },
            {
                Header: `K8S Version`,
                headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                accessor: 'status.orchestratorMetadata.version'
                // sortField: clusterSortFields.K8SVERSION
            },
            // TODO: enable this column after data is available from the API
            // {
            //     Header: `Created`,
            //     headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            //     className: `w-1/8 ${defaultColumnClassName}`,
            //     accessor: 'createdAt',
            //     sortField: clusterSortFields.CREATED
            // },
            {
                Header: `Namespaces`,
                entityType: entityTypes.NAMESPACE,
                headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                // eslint-disable-next-line
                Cell: ({ original, pdf }) => (
                    <TableCountLink
                        entityType={entityTypes.NAMESPACE}
                        count={original.namespaceCount}
                        textOnly={pdf}
                        selectedRowId={original.id}
                    />
                ),
                accessor: 'namespaceCount'
                // sortField: clusterSortFields.NAMESPACE
            },
            {
                Header: `Deployments`,
                entityType: entityTypes.DEPLOYMENT,
                headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                // eslint-disable-next-line
                Cell: ({ original, pdf }) => (
                    <TableCountLink
                        entityType={entityTypes.DEPLOYMENT}
                        count={original.deploymentCount}
                        textOnly={pdf}
                        selectedRowId={original.id}
                    />
                ),
                id: 'deploymentCount',
                accessor: 'deploymentCount'
                // sortField: clusterSortFields.DEPLOYMENTS
            },
            {
                Header: `Policies`,
                entityType: entityTypes.POLICY,
                headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                // eslint-disable-next-line
                Cell: ({ original, pdf }) => (
                    <TableCountLink
                        entityType={entityTypes.POLICY}
                        count={original.policyCount}
                        textOnly={pdf}
                        selectedRowId={original.id}
                    />
                ),
                id: 'policyCount',
                accessor: 'policyCount'
                // sortField: clusterSortFields.POLICIES
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
                accessor: 'policyStatus.status'
                // sortField: clusterSortFields.POLICY_STATUS
            },
            {
                Header: `Latest Violation`,
                headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => {
                    const { latestViolation } = original;
                    return <DateTimeField date={latestViolation} asString={pdf} />;
                },
                accessor: 'latestViolation'
                // sortField: clusterSortFields.LATEST_VIOLATION
            },
            {
                Header: `Risk Priority`,
                headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                accessor: 'priority'
                // sortField: clusterSortFields.PRIORITY
            }
        ];
        return removeEntityContextColumns(tableColumns, workflowState);
    }

    return (
        <WorkflowListPage
            data={data}
            query={query}
            queryOptions={queryOptions}
            entityListType={entityTypes.CLUSTER}
            getTableColumns={getTableColumns}
            selectedRowId={selectedRowId}
            search={search}
            page={page}
        />
    );
};

VulnMgmtClusters.propTypes = workflowListPropTypes;
VulnMgmtClusters.defaultProps = {
    ...workflowListDefaultProps,
    sort: null
};

export default VulnMgmtClusters;
