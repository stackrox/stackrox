import React from 'react';
import gql from 'graphql-tag';

import queryService from 'modules/queryService';
import TopCvssLabel from 'Components/TopCvssLabel';
import TableCountLink from 'Components/workflow/TableCountLink';
import StatusChip from 'Components/StatusChip';
import CVEStackedPill from 'Components/CVEStackedPill';
import DateTimeField from 'Components/DateTimeField';
import {
    defaultHeaderClassName,
    nonSortableHeaderClassName,
    defaultColumnClassName
} from 'Components/Table';
import entityTypes from 'constants/entityTypes';
import { LIST_PAGE_SIZE } from 'constants/workflowPages.constants';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';
import { IMAGE_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';
import removeEntityContextColumns from 'utils/tableUtils';
import { imageSortFields } from 'constants/sortFields';

export const defaultImageSort = [
    {
        id: imageSortFields.PRIORITY,
        desc: false
    },
    {
        id: imageSortFields.NAME,
        desc: false
    }
];

export function getImageTableColumns(workflowState) {
    const tableColumns = [
        {
            Header: 'Id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'id'
        },
        {
            Header: `Image`,
            headerClassName: `w-1/6 ${defaultHeaderClassName}`,
            className: `w-1/6 word-break-all ${defaultColumnClassName}`,
            accessor: 'name.fullName',
            sortField: imageSortFields.NAME
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
            sortField: imageSortFields.CVE_COUNT
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
            accessor: 'topVuln.cvss',
            sortField: imageSortFields.TOP_CVSS
        },
        {
            Header: `Created`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { metadata } = original;
                if (!metadata || !metadata.v1) return '–';
                return <DateTimeField date={metadata.v1.created} asString={pdf} />;
            },
            accessor: 'metadata.v1.created',
            sortField: imageSortFields.CREATED_TIME
        },
        {
            Header: `Scan Time`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { scan } = original;
                if (!scan) return '–';
                return <DateTimeField date={scan.scanTime} asString={pdf} />;
            },
            accessor: 'scan.scanTime',
            sortField: imageSortFields.SCAN_TIME
        },
        {
            Header: 'Image Status',
            headerClassName: `w-1/10 ${nonSortableHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { deploymentCount } = original;
                const imageStatus = deploymentCount === 0 ? 'inactive' : 'active';
                return <StatusChip status={imageStatus} asString={pdf} />;
            },
            accessor: 'deploymentCount',
            sortField: imageSortFields.IMAGE_STATUS,
            sortable: false
        },
        {
            Header: `Deployments`,
            entityType: entityTypes.DEPLOYMENT,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => (
                <TableCountLink
                    entityType={entityTypes.DEPLOYMENT}
                    count={original.deploymentCount}
                    textOnly={pdf}
                    selectedRowId={original.id}
                />
            ),
            accessor: 'deploymentCount',
            sortField: imageSortFields.DEPLOYMENT_COUNT
        },
        {
            Header: `Components`,
            entityType: entityTypes.COMPONENT,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { scan, id } = original;
                if (!scan) return '-';
                const { components } = scan;
                return (
                    <TableCountLink
                        entityType={entityTypes.COMPONENT}
                        count={components.length}
                        textOnly={pdf}
                        selectedRowId={id}
                    />
                );
            },
            accessor: 'scan.components',
            sortField: imageSortFields.COMPONENT_COUNT
        },
        {
            Header: `Risk Priority`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            accessor: 'priority',
            sortField: imageSortFields.PRIORITY
        }
    ];
    return removeEntityContextColumns(tableColumns, workflowState);
}

const VulnMgmtImages = ({ selectedRowId, search, sort, page, data, totalResults }) => {
    const query = gql`
        query getImages($query: String, $pagination: Pagination) {
            results: images(query: $query, pagination: $pagination) {
                ...imageFields
            }
            count: imageCount(query: $query)
        }
        ${IMAGE_LIST_FRAGMENT}
    `;

    const tableSort = sort || defaultImageSort;
    const queryOptions = {
        variables: {
            query: queryService.objectToWhereClause(search),
            pagination: queryService.getPagination(tableSort, page, LIST_PAGE_SIZE)
        }
    };

    return (
        <WorkflowListPage
            data={data}
            totalResults={totalResults}
            query={query}
            queryOptions={queryOptions}
            entityListType={entityTypes.IMAGE}
            getTableColumns={getImageTableColumns}
            selectedRowId={selectedRowId}
            search={search}
            page={page}
        />
    );
};

VulnMgmtImages.propTypes = workflowListPropTypes;
VulnMgmtImages.defaultProps = workflowListDefaultProps;

export default VulnMgmtImages;
