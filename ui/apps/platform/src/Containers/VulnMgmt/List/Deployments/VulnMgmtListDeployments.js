import React from 'react';
import { gql } from '@apollo/client';

import queryService from 'utils/queryService';
import DateTimeField from 'Components/DateTimeField';
import CVEStackedPill from 'Components/CVEStackedPill';
import TableCellLink from 'Components/TableCellLink';
import TableCountLink from 'Components/workflow/TableCountLink';
import {
    defaultHeaderClassName,
    nonSortableHeaderClassName,
    defaultColumnClassName,
} from 'Components/Table';
import entityTypes from 'constants/entityTypes';
import { LIST_PAGE_SIZE } from 'constants/workflowPages.constants';
import { DEPLOYMENT_LIST_FRAGMENT_UPDATED } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';
import removeEntityContextColumns from 'utils/tableUtils';
import { deploymentSortFields } from 'constants/sortFields';
import { getRatioOfScannedImages } from './deployments.utils';
import WorkflowListPage from '../WorkflowListPage';
import { vulMgmtPolicyQuery } from '../../Entity/VulnMgmtPolicyQueryUtil';

export const defaultDeploymentSort = [
    {
        id: deploymentSortFields.PRIORITY,
        desc: false,
    },
];

export function getCurriedDeploymentTableColumns() {
    return function getDeploymentTableColumns(workflowState) {
        // to determine whether to show the counts as links in the table when not in pure DEPLOYMENT state
        const inFindingsSection =
            workflowState.getCurrentEntity().entityType !== entityTypes.DEPLOYMENT;

        const tableColumns = [
            {
                Header: 'Id',
                headerClassName: 'hidden',
                className: 'hidden',
                accessor: 'id',
            },
            {
                Header: `Deployment`,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                id: deploymentSortFields.DEPLOYMENT,
                accessor: 'name',
                sortField: deploymentSortFields.DEPLOYMENT,
            },
            {
                Header: `Image CVEs`,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                entityType: entityTypes.IMAGE_CVE,
                Cell: ({ original, pdf }) => {
                    const { imageVulnerabilityCounter, id, images } = original;
                    if (
                        !imageVulnerabilityCounter ||
                        (imageVulnerabilityCounter.all && imageVulnerabilityCounter.all.total === 0)
                    ) {
                        const scanRatio = getRatioOfScannedImages(images);

                        if (!scanRatio.scanned && !scanRatio.total) {
                            return `No images scanned`;
                        }
                        if (scanRatio.scanned !== scanRatio.total) {
                            return `${scanRatio.scanned || 0} / ${
                                scanRatio.total || 0
                            } images scanned`;
                        }
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
                            hideLink={pdf || inFindingsSection}
                        />
                    );
                },
                id: deploymentSortFields.CVE_COUNT,
                accessor: 'imageVulnerabilityCounter.all.total',
                sortField: deploymentSortFields.CVE_COUNT,
            },
            {
                Header: `Latest Violation`,
                headerClassName: `w-1/8 ${nonSortableHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => {
                    const { latestViolation } = original;
                    return <DateTimeField date={latestViolation} asString={pdf} />;
                },
                id: deploymentSortFields.LATEST_VIOLATION,
                accessor: 'latestViolation',
                sortField: deploymentSortFields.LATEST_VIOLATION,
                sortable: false,
            },
            {
                Header: `Cluster`,
                entityType: entityTypes.CLUSTER,
                headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => {
                    const { clusterName, clusterId, id } = original;
                    const url = workflowState
                        .pushListItem(id)
                        .pushRelatedEntity(entityTypes.CLUSTER, clusterId)
                        .toUrl();
                    return (
                        <TableCellLink pdf={inFindingsSection || pdf} url={url}>
                            {clusterName}
                        </TableCellLink>
                    );
                },
                id: deploymentSortFields.CLUSTER,
                accessor: 'clusterName',
                sortField: deploymentSortFields.CLUSTER,
            },
            {
                Header: `Namespace`,
                entityType: entityTypes.NAMESPACE,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => {
                    const { namespace, namespaceId, id } = original;
                    const url = workflowState
                        .pushListItem(id)
                        .pushRelatedEntity(entityTypes.NAMESPACE, namespaceId)
                        .toUrl();
                    return (
                        <TableCellLink pdf={inFindingsSection || pdf} url={url}>
                            {namespace}
                        </TableCellLink>
                    );
                },
                id: deploymentSortFields.NAMESPACE,
                accessor: 'namespace',
                sortField: deploymentSortFields.NAMESPACE,
            },
            {
                Header: `Images`,
                headerClassName: `w-1/10 ${nonSortableHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => (
                    <TableCountLink
                        entityType={entityTypes.IMAGE}
                        count={original.imageCount}
                        textOnly={inFindingsSection || pdf}
                        selectedRowId={original.id}
                    />
                ),
                id: deploymentSortFields.IMAGE_COUNT,
                accessor: 'imageCount',
                // TODO: restore sorting on this field, see https://issues.redhat.com/browse/ROX-12548 for context
                // sortField: componentSortFields.IMAGES,
                sortable: false,
            },
            {
                Header: `Risk Priority`,
                headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                id: deploymentSortFields.PRIORITY,
                accessor: 'priority',
                sortField: deploymentSortFields.PRIORITY,
            },
        ];

        return removeEntityContextColumns(tableColumns, workflowState);
    };
}

const VulnMgmtDeployments = ({ selectedRowId, search, sort, page, data, totalResults }) => {
    const query = gql`
        query getDeployments($query: String, $policyQuery: String, $pagination: Pagination) {
            results: deployments(query: $query, pagination: $pagination) {
                ...deploymentFields
            }
            count: deploymentCount(query: $query)
        }
        ${DEPLOYMENT_LIST_FRAGMENT_UPDATED}
    `;
    const tableSort = sort || defaultDeploymentSort;
    const queryOptions = {
        variables: {
            query: queryService.objectToWhereClause(search),
            ...vulMgmtPolicyQuery,
            pagination: queryService.getPagination(tableSort, page, LIST_PAGE_SIZE),
        },
    };

    const getDeploymentTableColumns = getCurriedDeploymentTableColumns();

    return (
        <WorkflowListPage
            data={data}
            totalResults={totalResults}
            query={query}
            queryOptions={queryOptions}
            idAttribute="id"
            entityListType={entityTypes.DEPLOYMENT}
            getTableColumns={getDeploymentTableColumns}
            selectedRowId={selectedRowId}
            search={search}
            sort={tableSort}
            page={page}
        />
    );
};

VulnMgmtDeployments.propTypes = workflowListPropTypes;
VulnMgmtDeployments.defaultProps = workflowListDefaultProps;

export default VulnMgmtDeployments;
