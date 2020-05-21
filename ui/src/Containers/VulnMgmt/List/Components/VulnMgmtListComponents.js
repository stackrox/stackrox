import React from 'react';
import gql from 'graphql-tag';

import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import TopCvssLabel from 'Components/TopCvssLabel';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';
import entityTypes from 'constants/entityTypes';
import { LIST_PAGE_SIZE } from 'constants/workflowPages.constants';
import CVEStackedPill from 'Components/CVEStackedPill';
import TableCountLink from 'Components/workflow/TableCountLink';
import queryService from 'utils/queryService';

import { VULN_COMPONENT_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';
import removeEntityContextColumns from 'utils/tableUtils';
import { componentSortFields } from 'constants/sortFields';

import { getFilteredComponentColumns } from './ListComponents.utils';

export const defaultComponentSort = [
    {
        id: componentSortFields.PRIORITY,
        desc: false,
    },
];

export function getComponentTableColumns(workflowState) {
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
            Header: `CVEs`,
            entityType: entityTypes.CVE,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { vulnCounter, id } = original;
                if (!vulnCounter || vulnCounter.all.total === 0) return 'No CVEs';

                const newState = workflowState.pushListItem(id).pushList(entityTypes.CVE);
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
            Header: `Top CVSS`,
            headerClassName: `w-1/10 text-center ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { topVuln } = original;
                if (!topVuln)
                    return (
                        <div className="mx-auto flex flex-col">
                            <span>–</span>
                        </div>
                    );
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
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
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
            sortField: componentSortFields.IMAGE_COUNT,
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

    const componentColumnsBasedOnContext = getFilteredComponentColumns(tableColumns, workflowState);

    return removeEntityContextColumns(componentColumnsBasedOnContext, workflowState);
}

const VulnMgmtComponents = ({ selectedRowId, search, sort, page, data, totalResults }) => {
    const query = gql`
        query getComponents($query: String, $pagination: Pagination) {
            results: components(query: $query, pagination: $pagination) {
                ...componentFields
            }
            count: componentCount(query: $query)
        }
        ${VULN_COMPONENT_LIST_FRAGMENT}
    `;
    const tableSort = sort || defaultComponentSort;
    const queryOptions = {
        variables: {
            query: queryService.objectToWhereClause(search),
            scopeQuery: '',
            pagination: queryService.getPagination(tableSort, page, LIST_PAGE_SIZE),
        },
    };

    return (
        <WorkflowListPage
            data={data}
            totalResults={totalResults}
            query={query}
            queryOptions={queryOptions}
            idAttribute="id"
            entityListType={entityTypes.COMPONENT}
            getTableColumns={getComponentTableColumns}
            selectedRowId={selectedRowId}
            search={search}
            sort={tableSort}
            page={page}
        />
    );
};

VulnMgmtComponents.propTypes = workflowListPropTypes;
VulnMgmtComponents.defaultProps = {
    ...workflowListDefaultProps,
    sort: null,
};

export default VulnMgmtComponents;
