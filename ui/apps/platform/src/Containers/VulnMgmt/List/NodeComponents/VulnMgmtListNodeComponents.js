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
import TableCountLink from 'Components/workflow/TableCountLink';
import queryService from 'utils/queryService';

import { VULN_NODE_COMPONENT_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';
import removeEntityContextColumns from 'utils/tableUtils';
import { componentSortFields } from 'constants/sortFields';

import { getFilteredComponentColumns } from './ListNodeComponents.utils';
import WorkflowListPage from '../WorkflowListPage';

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
                Cell: ({ original }) => {
                    const { version, name } = original;
                    return `${name} ${version}`;
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
                Header: `Node CVEs`,
                entityType: entityTypes.NODE_CVE,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => {
                    const { vulnCounter, id } = original;
                    if (!vulnCounter || vulnCounter.all.total === 0) {
                        return 'No CVEs';
                    }

                    const newState = workflowState.pushListItem(id).pushList(entityTypes.NODE_CVE);
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
                Header: `Nodes`,
                entityType: entityTypes.NODE,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                id: componentSortFields.NODE_COUNT,
                accessor: 'nodeCount',
                Cell: ({ original, pdf }) => (
                    <TableCountLink
                        entityType={entityTypes.NODE}
                        count={original.nodeCount}
                        textOnly={pdf}
                        selectedRowId={original.id}
                    />
                ),
                sortField: componentSortFields.NODE_COUNT,
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

const VulnMgmtNodeComponents = ({ selectedRowId, search, sort, page, data, totalResults }) => {
    const query = gql`
        query getNodeComponents($query: String, $pagination: Pagination) {
            results: nodeComponents(query: $query, pagination: $pagination) {
                ...nodeComponentFields
            }
            count: nodeComponentCount(query: $query)
        }
        ${VULN_NODE_COMPONENT_LIST_FRAGMENT}
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
            entityListType={entityTypes.NODE_COMPONENT}
            getTableColumns={getTableColumns}
            selectedRowId={selectedRowId}
            search={search}
            sort={tableSort}
            page={page}
        />
    );
};

VulnMgmtNodeComponents.propTypes = workflowListPropTypes;
VulnMgmtNodeComponents.defaultProps = workflowListDefaultProps;

export default VulnMgmtNodeComponents;
