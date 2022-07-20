import React from 'react';
import { gql } from '@apollo/client';

import queryService from 'utils/queryService';
import TableCellLink from 'Components/TableCellLink';
import TableCountLink from 'Components/workflow/TableCountLink';
import CVEStackedPill from 'Components/CVEStackedPill';
import StatusChip from 'Components/StatusChip';
import DateTimeField from 'Components/DateTimeField';
import {
    defaultHeaderClassName,
    nonSortableHeaderClassName,
    defaultColumnClassName,
} from 'Components/Table';
import entityTypes from 'constants/entityTypes';
import { LIST_PAGE_SIZE } from 'constants/workflowPages.constants';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';
import {
    NAMESPACE_LIST_FRAGMENT,
    NAMESPACE_LIST_FRAGMENT_UPDATED,
} from 'Containers/VulnMgmt/VulnMgmt.fragments';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';
import useFeatureFlags from 'hooks/useFeatureFlags';
import removeEntityContextColumns from 'utils/tableUtils';
import { namespaceSortFields } from 'constants/sortFields';
import { vulMgmtPolicyQuery } from '../../Entity/VulnMgmtPolicyQueryUtil';

export const defaultNamespaceSort = [
    {
        id: namespaceSortFields.PRIORITY,
        desc: false,
    },
];

const VulnMgmtNamespaces = ({ selectedRowId, search, sort, page, data, totalResults }) => {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isFrontendVMUpdatesEnabled = isFeatureFlagEnabled('ROX_FRONTEND_VM_UPDATES');

    const fragmentToUse = isFrontendVMUpdatesEnabled
        ? NAMESPACE_LIST_FRAGMENT_UPDATED
        : NAMESPACE_LIST_FRAGMENT;

    function getNamespaceTableColumns(workflowState) {
        const tableColumns = [
            {
                Header: 'Id',
                headerClassName: 'hidden',
                className: 'hidden',
                accessor: 'metadata.id',
            },
            {
                Header: `Namespace`,
                headerClassName: `w-1/6 ${defaultHeaderClassName}`,
                className: `w-1/6 ${defaultColumnClassName}`,
                id: namespaceSortFields.NAMESPACE,
                accessor: 'metadata.name',
                sortField: namespaceSortFields.NAMESPACE,
            },
            {
                Header: `CVEs`,
                entityType: entityTypes.CVE,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => {
                    const { vulnCounter, metadata } = original;
                    if (!vulnCounter || vulnCounter.all.total === 0) {
                        return 'No CVEs';
                    }

                    const newState = workflowState
                        .pushListItem(metadata.id)
                        .pushList(entityTypes.CVE);
                    const cvesUrl = newState.toUrl();
                    const fixableUrl = newState.setSearch({ Fixable: true }).toUrl();

                    return (
                        <CVEStackedPill
                            vulnCounter={vulnCounter}
                            url={cvesUrl}
                            fixableUrl={fixableUrl}
                            hideLink={pdf}
                        />
                    );
                },
                id: namespaceSortFields.CVE_COUNT,
                accessor: 'vulnCounter.all.total',
                sortField: namespaceSortFields.CVE_COUNT,
            },
            {
                Header: `Image CVEs`,
                entityType: entityTypes.IMAGE_CVE,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => {
                    const { imageVulnerabilityCounter, metadata } = original;
                    if (!imageVulnerabilityCounter || imageVulnerabilityCounter.all.total === 0) {
                        return 'No CVEs';
                    }
                    const newState = workflowState
                        .pushListItem(metadata.id)
                        .pushList(entityTypes.IMAGE_CVE);
                    const cvesUrl = newState.toUrl();
                    const fixableUrl = newState.setSearch({ Fixable: true }).toUrl();

                    return (
                        <CVEStackedPill
                            vulnCounter={imageVulnerabilityCounter}
                            url={cvesUrl}
                            fixableUrl={fixableUrl}
                            hideLink={pdf}
                        />
                    );
                },
                id: namespaceSortFields.CVE_COUNT,
                accessor: 'vulnCounter.all.total',
                sortField: namespaceSortFields.CVE_COUNT,
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

                    return (
                        <TableCellLink pdf={pdf} url={url}>
                            {clusterName}
                        </TableCellLink>
                    );
                },
                id: namespaceSortFields.CLUSTER,
                accessor: 'metadata.clusterName',
                sortField: namespaceSortFields.CLUSTER,
            },
            {
                Header: `Deployments`,
                entityType: entityTypes.DEPLOYMENT,
                headerClassName: `w-1/8 ${nonSortableHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => (
                    <TableCountLink
                        entityType={entityTypes.DEPLOYMENT}
                        count={original.deploymentCount}
                        textOnly={pdf}
                        selectedRowId={original.metadata.id}
                    />
                ),
                id: namespaceSortFields.DEPLOYMENT_COUNT,
                accessor: 'deploymentCount',
                sortField: namespaceSortFields.DEPLOYMENT_COUNT,
                sortable: false,
            },
            {
                Header: `Images`,
                entityType: entityTypes.IMAGE,
                headerClassName: `w-1/8 ${nonSortableHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => (
                    <TableCountLink
                        entityType={entityTypes.IMAGE}
                        count={original.imageCount}
                        textOnly={pdf}
                        selectedRowId={original.metadata.id}
                    />
                ),
                id: namespaceSortFields.IMAGES,
                accessor: 'imageCount',
                sortField: namespaceSortFields.IMAGES,
                sortable: false,
            },
            // @TODD, restore the Policy Counts column once its performance is improved,
            //   or remove the comment if we determine that it cannot be made performant
            //   (see https://stack-rox.atlassian.net/browse/ROX-4080)
            // {
            //     Header: `Policies`,
            //     entityType: entityTypes.POLICY,
            //     headerClassName: `w-1/8 ${nonSortableHeaderClassName}`,
            //     className: `w-1/8 ${defaultColumnClassName}`,
            //     Cell: ({ original, pdf }) => (
            //         <TableCountLink
            //             entityType={entityTypes.POLICY}
            //             count={original.policyCount}
            //             textOnly={pdf}
            //             selectedRowId={original.metadata.id}
            //         />
            //     ),
            //     id: namespaceSortFields.POLICY_COUNT,
            //     accessor: 'policyCount',
            //     sortField: namespaceSortFields.POLICY_COUNT,
            //     sortable: false
            // },
            {
                Header: `Policy Status`,
                headerClassName: `w-1/10 ${nonSortableHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => {
                    const { policyStatusOnly } = original;
                    const policyLabel = <StatusChip status={policyStatusOnly} asString={pdf} />;

                    return policyLabel;
                },
                id: namespaceSortFields.POLICY_STATUS,
                accessor: 'policyStatusOnly',
                sortField: namespaceSortFields.POLICY_STATUS,
                sortable: false,
            },
            {
                Header: `Latest Violation`,
                headerClassName: `w-1/8 ${nonSortableHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => {
                    const { latestViolation } = original;
                    return <DateTimeField date={latestViolation} asString={pdf} />;
                },
                id: namespaceSortFields.LATEST_VIOLATION,
                accessor: 'latestViolation',
                sortField: namespaceSortFields.LATEST_VIOLATION,
                sortable: false,
            },
            {
                Header: `Risk Priority`,
                headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                id: namespaceSortFields.PRIORITY,
                accessor: 'metadata.priority',
                sortField: namespaceSortFields.PRIORITY,
                sortable: true,
            },
        ];

        const flagGatedTableColumns = tableColumns.filter((col) => {
            if (isFrontendVMUpdatesEnabled) {
                if (col.Header === 'CVEs') {
                    return false;
                }
            } else if (col.Header === 'Image CVEs') {
                return false;
            }
            return true;
        });
        return removeEntityContextColumns(flagGatedTableColumns, workflowState);
    }

    const query = gql`
        query getNamespaces($query: String, $policyQuery: String, $pagination: Pagination) {
            results: namespaces(query: $query, pagination: $pagination) {
                ...namespaceFields
                unusedVarSink(query: $policyQuery)
            }
            count: namespaceCount(query: $query)
        }
        ${fragmentToUse}
    `;
    const tableSort = sort || defaultNamespaceSort;
    const queryOptions = {
        variables: {
            query: queryService.objectToWhereClause(search),
            ...vulMgmtPolicyQuery,
            pagination: queryService.getPagination(tableSort, page, LIST_PAGE_SIZE),
        },
    };

    return (
        <WorkflowListPage
            data={data}
            totalResults={totalResults}
            query={query}
            queryOptions={queryOptions}
            entityListType={entityTypes.NAMESPACE}
            getTableColumns={getNamespaceTableColumns}
            selectedRowId={selectedRowId}
            idAttribute="metadata.id"
            search={search}
            sort={tableSort}
            page={page}
        />
    );
};

VulnMgmtNamespaces.propTypes = workflowListPropTypes;
VulnMgmtNamespaces.defaultProps = workflowListDefaultProps;

export default VulnMgmtNamespaces;
