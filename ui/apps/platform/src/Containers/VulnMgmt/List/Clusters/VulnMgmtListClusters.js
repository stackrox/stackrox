import React from 'react';
import { gql } from '@apollo/client';

import queryService from 'utils/queryService';
import {
    defaultHeaderClassName,
    nonSortableHeaderClassName,
    defaultColumnClassName,
} from 'Components/Table';
import DateTimeField from 'Components/DateTimeField';
import ClusterTableCountLinks from 'Components/workflow/ClusterTableCountLinks';
import entityTypes from 'constants/entityTypes';
import CVEStackedPill from 'Components/CVEStackedPill';

import { CLUSTER_LIST_FRAGMENT_UPDATED } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';
import { clusterSortFields } from 'constants/sortFields';
import { LIST_PAGE_SIZE } from 'constants/workflowPages.constants';
import removeEntityContextColumns from 'utils/tableUtils';
import { vulMgmtPolicyQuery } from '../../Entity/VulnMgmtPolicyQueryUtil';
import WorkflowListPage from '../WorkflowListPage';

export const defaultClusterSort = [
    {
        id: clusterSortFields.PRIORITY,
        desc: false,
    },
];

const VulnMgmtClusters = ({ selectedRowId, search, sort, page, data, totalResults }) => {
    const query = gql`
        query getClusters(
            $query: String
            $policyQuery: String
            $scopeQuery: String
            $pagination: Pagination
        ) {
            results: clusters(query: $query, pagination: $pagination) {
                ...clusterFields
                unusedVarSink(query: $policyQuery)
                unusedVarSink(query: $scopeQuery)
            }
            count: clusterCount(query: $query)
        }
        ${CLUSTER_LIST_FRAGMENT_UPDATED}
    `;

    const tableSort = sort || defaultClusterSort;
    const queryOptions = {
        variables: {
            ...vulMgmtPolicyQuery,
            query: queryService.objectToWhereClause(search),
            pagination: queryService.getPagination(tableSort, page, LIST_PAGE_SIZE),
        },
    };

    function getTableColumns(workflowState) {
        const tableColumns = [
            {
                Header: 'Id',
                headerClassName: 'hidden',
                className: 'hidden',
                accessor: 'id',
            },
            {
                Header: `Cluster`,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                id: clusterSortFields.CLUSTER,
                accessor: 'name',
                sortField: clusterSortFields.CLUSTER,
            },
            {
                Header: `Image CVEs`,
                entityType: entityTypes.IMAGE_CVE,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => {
                    const { imageVulnerabilityCounter, id } = original;
                    if (!imageVulnerabilityCounter || imageVulnerabilityCounter.all.total === 0) {
                        return 'No CVEs';
                    }

                    const newState = workflowState.pushListItem(id).pushList(entityTypes.IMAGE_CVE);
                    const url = newState.toUrl();
                    const fixableUrl = newState.setSearch({ Fixable: true }).toUrl();

                    return (
                        <CVEStackedPill
                            vulnCounter={imageVulnerabilityCounter}
                            url={url}
                            fixableUrl={fixableUrl}
                            hideLink={pdf}
                        />
                    );
                },
                id: clusterSortFields.CVE_COUNT,
                accessor: 'vulnCounter.all.total',
                sortField: clusterSortFields.CVE_COUNT,
            },
            {
                Header: `Node CVEs`,
                entityType: entityTypes.NODE_CVE,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => {
                    const { nodeVulnerabilityCounter, id } = original;
                    if (!nodeVulnerabilityCounter || nodeVulnerabilityCounter.all.total === 0) {
                        return 'No CVEs';
                    }

                    const newState = workflowState.pushListItem(id).pushList(entityTypes.NODE_CVE);
                    const url = newState.toUrl();
                    const fixableUrl = newState.setSearch({ Fixable: true }).toUrl();

                    return (
                        <CVEStackedPill
                            vulnCounter={nodeVulnerabilityCounter}
                            url={url}
                            fixableUrl={fixableUrl}
                            hideLink={pdf}
                        />
                    );
                },
                id: clusterSortFields.CVE_COUNT,
                accessor: 'vulnCounter.all.total',
                sortField: clusterSortFields.CVE_COUNT,
            },
            {
                Header: `Platform CVEs`,
                entityType: entityTypes.CLUSTER_CVE,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => {
                    const { clusterVulnerabilityCounter, id } = original;
                    if (
                        !clusterVulnerabilityCounter ||
                        clusterVulnerabilityCounter.all.total === 0
                    ) {
                        return 'No CVEs';
                    }

                    const newState = workflowState
                        .pushListItem(id)
                        .pushList(entityTypes.CLUSTER_CVE);
                    const url = newState.toUrl();
                    const fixableUrl = newState.setSearch({ Fixable: true }).toUrl();

                    return (
                        <CVEStackedPill
                            vulnCounter={clusterVulnerabilityCounter}
                            url={url}
                            fixableUrl={fixableUrl}
                            hideLink={pdf}
                        />
                    );
                },
                id: clusterSortFields.CVE_COUNT,
                accessor: 'vulnCounter.all.total',
                sortField: clusterSortFields.CVE_COUNT,
            },
            {
                Header: `K8S Version`,
                headerClassName: `w-1/10 ${nonSortableHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                id: clusterSortFields.K8SVERSION,
                accessor: 'status.orchestratorMetadata.version',
                sortField: clusterSortFields.K8SVERSION,
                sortable: false,
            },
            // TODO: enable this column after data is available from the API
            // {
            //     Header: `Created`,
            //     headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            //     className: `w-1/8 ${defaultColumnClassName}`,
            //     id: clusterSortFields.CREATED,
            //     accessor: 'createdAt',
            //     sortField: clusterSortFields.CREATED
            // },
            {
                Header: `Entities`,
                headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => (
                    <ClusterTableCountLinks row={original} textOnly={pdf} />
                ),
                accessor: 'entities',
                sortable: false,
            },
            {
                Header: `Latest Violation`,
                headerClassName: `w-1/10 ${nonSortableHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => {
                    const { latestViolation } = original;
                    return <DateTimeField date={latestViolation} asString={pdf} />;
                },
                id: clusterSortFields.LATEST_VIOLATION,
                accessor: 'latestViolation',
                sortField: clusterSortFields.LATEST_VIOLATION,
                sortable: false,
            },
            {
                Header: `Risk Priority`,
                headerClassName: `w-1/10 ${nonSortableHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                id: clusterSortFields.PRIORITY,
                accessor: 'priority',
                sortField: clusterSortFields.PRIORITY,
                sortable: true,
            },
        ];

        return removeEntityContextColumns(tableColumns, workflowState);
    }

    return (
        <WorkflowListPage
            data={data}
            totalResults={totalResults}
            query={query}
            queryOptions={queryOptions}
            entityListType={entityTypes.CLUSTER}
            getTableColumns={getTableColumns}
            selectedRowId={selectedRowId}
            search={search}
            sort={tableSort}
            page={page}
        />
    );
};

VulnMgmtClusters.propTypes = workflowListPropTypes;
VulnMgmtClusters.defaultProps = workflowListDefaultProps;

export default VulnMgmtClusters;
