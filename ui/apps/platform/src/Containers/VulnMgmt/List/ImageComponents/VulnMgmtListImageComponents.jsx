import React from 'react';
import { gql } from '@apollo/client';

import {
    defaultHeaderClassName,
    defaultColumnClassName,
    nonSortableHeaderClassName,
} from 'Components/Table';
import TopCvssLabel from 'Components/TopCvssLabel';
import entityTypes from 'constants/entityTypes';
import { LIST_PAGE_SIZE } from 'constants/workflowPages.constants';
import CVEStackedPill from 'Components/CVEStackedPill';
import queryService from 'utils/queryService';

import { VULN_IMAGE_COMPONENT_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';
import removeEntityContextColumns from 'utils/tableUtils';
import { componentSortFields } from 'constants/sortFields';

import TableCellLink from 'Components/TableCellLink';
import TableCountLink from 'Components/workflow/TableCountLink';
import { getFilteredComponentColumns } from './ListImageComponents.utils';
import WorkflowListPage from '../WorkflowListPage';
import { getVulnMgmtPathForEntitiesAndId } from '../../VulnMgmt.utils/entities';

export const defaultComponentSort = [
    {
        id: componentSortFields.PRIORITY,
        desc: false,
    },
];

export function getComponentTableColumns() {
    return function getTableColumns(workflowState, isFeatureFlagEnabled) {
        const tableColumns = [
            {
                Header: 'Id',
                headerClassName: 'hidden',
                className: 'hidden',
                accessor: 'id',
            },
            {
                Header: `Component`,
                headerClassName: `w-1/4 ${defaultHeaderClassName}`,
                className: `w-1/4 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => {
                    const url = getVulnMgmtPathForEntitiesAndId('IMAGE_COMPONENT', original.id);
                    return (
                        <TableCellLink pdf={pdf} url={url}>
                            {`${original.name} ${original.version}`}
                        </TableCellLink>
                    );
                },
                id: componentSortFields.COMPONENT,
                accessor: 'name',
                sortField: componentSortFields.COMPONENT,
            },
            {
                Header: `Operating System`,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                id: componentSortFields.OPERATING_SYSTEM,
                accessor: 'operatingSystem',
                sortField: componentSortFields.OPERATING_SYSTEM,
            },
            {
                Header: `Image CVEs`,
                entityType: entityTypes.IMAGE_CVE,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => {
                    const { vulnCounter, id } = original;
                    if (!vulnCounter || vulnCounter.all.total === 0) {
                        return 'No CVEs';
                    }

                    const newState = workflowState.pushListItem(id).pushList(entityTypes.IMAGE_CVE);
                    const url = newState.toUrl();
                    const fixableUrl = newState.setSearch({ Fixable: true }).toUrl();

                    return (
                        <CVEStackedPill
                            vulnCounter={vulnCounter}
                            url={url}
                            fixableUrl={fixableUrl}
                            hideLink={pdf}
                        />
                    );
                },
                id: componentSortFields.CVE_COUNT,
                accessor: 'vulnCounter.all.total',
                sortField: componentSortFields.CVE_COUNT,
            },
            {
                Header: `Active`,
                headerClassName: `w-1/10 text-center ${nonSortableHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                Cell: ({ original }) => {
                    return original.activeState?.state || 'Undetermined';
                },
                id: componentSortFields.ACTIVE,
                accessor: 'isActive',
                sortField: componentSortFields.ACTIVE,
                sortable: false,
            },
            {
                Header: `Fixed In`,
                headerClassName: `w-1/12 ${defaultHeaderClassName}`,
                className: `w-1/12 word-break-all ${defaultColumnClassName}`,
                Cell: ({ original }) =>
                    original.fixedIn ||
                    (original.vulnCounter.all.total === 0 ? 'N/A' : 'Not Fixable'),
                id: componentSortFields.FIXEDIN,
                accessor: 'fixedIn',
                sortField: componentSortFields.FIXEDIN,
                sortable: false,
            },
            {
                Header: `Top CVSS`,
                headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                Cell: ({ original }) => {
                    const { topVuln } = original;
                    if (!topVuln) {
                        return 'N/A';
                    }
                    const { cvss, scoreVersion } = topVuln;
                    return <TopCvssLabel cvss={cvss} version={scoreVersion} />;
                },
                id: componentSortFields.TOP_CVSS,
                accessor: 'topVuln.cvss',
                sortField: componentSortFields.TOP_CVSS,
            },
            {
                Header: `Source`,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                id: componentSortFields.SOURCE,
                accessor: 'source',
                sortField: componentSortFields.SOURCE,
            },
            {
                Header: `Location`,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 word-break-all ${defaultColumnClassName}`,
                Cell: ({ original }) => original.location || 'N/A',
                id: componentSortFields.LOCATION,
                accessor: 'location',
                sortField: componentSortFields.LOCATION,
            },
            {
                Header: `Images`,
                entityType: entityTypes.IMAGE,
                headerClassName: `w-1/8 ${nonSortableHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                id: componentSortFields.IMAGE_COUNT,
                accessor: 'imageCount',
                Cell: ({ original, pdf }) => (
                    <TableCountLink
                        entityType={entityTypes.IMAGE}
                        count={original.imageCount}
                        textOnly={pdf}
                        selectedRowId={original.id}
                    />
                ),
                // TODO: restore sorting on this field, see https://issues.redhat.com/browse/ROX-12548 for context
                // sortField: componentSortFields.IMAGES,
                sortable: false,
            },
            {
                Header: `Deployments`,
                entityType: entityTypes.DEPLOYMENT,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                id: componentSortFields.DEPLOYMENT_COUNT,
                accessor: 'deploymentCount',
                Cell: ({ original, pdf }) => (
                    <TableCountLink
                        entityType={entityTypes.DEPLOYMENT}
                        count={original.deploymentCount}
                        textOnly={pdf}
                        selectedRowId={original.id}
                    />
                ),
                sortField: componentSortFields.DEPLOYMENT_COUNT,
            },
            {
                Header: `Risk Priority`,
                headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                id: componentSortFields.PRIORITY,
                accessor: 'priority',
                sortField: componentSortFields.PRIORITY,
            },
        ];

        const componentColumnsBasedOnContext = getFilteredComponentColumns(
            tableColumns,
            workflowState,
            isFeatureFlagEnabled
        );

        return removeEntityContextColumns(componentColumnsBasedOnContext, workflowState);
    };
}

const VulnMgmtListImageComponents = ({ selectedRowId, search, sort, page, data, totalResults }) => {
    const query = gql`
        query getImageComponents($query: String, $pagination: Pagination) {
            results: imageComponents(query: $query, pagination: $pagination) {
                ...imageComponentFields
            }
            count: imageComponentCount(query: $query)
        }
        ${VULN_IMAGE_COMPONENT_LIST_FRAGMENT}
    `;
    const tableSort = sort || defaultComponentSort;
    const queryOptions = {
        variables: {
            query: queryService.objectToWhereClause(search),
            scopeQuery: '',
            pagination: queryService.getPagination(tableSort, page, LIST_PAGE_SIZE),
        },
    };

    const getTableColumns = getComponentTableColumns();

    return (
        <WorkflowListPage
            data={data}
            totalResults={totalResults}
            query={query}
            queryOptions={queryOptions}
            idAttribute="id"
            entityListType={entityTypes.IMAGE_COMPONENT}
            getTableColumns={getTableColumns}
            selectedRowId={selectedRowId}
            search={search}
            sort={tableSort}
            page={page}
        />
    );
};

VulnMgmtListImageComponents.propTypes = workflowListPropTypes;
VulnMgmtListImageComponents.defaultProps = workflowListDefaultProps;

export default VulnMgmtListImageComponents;
