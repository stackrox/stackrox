import React from 'react';
import gql from 'graphql-tag';
import pluralize from 'pluralize';

import queryService from 'modules/queryService';
import TopCvssLabel from 'Components/TopCvssLabel';
import TableCellLink from 'Components/TableCellLink';
import StatusChip from 'Components/StatusChip';
import CVEStackedPill from 'Components/CVEStackedPill';
import DateTimeField from 'Components/DateTimeField';
import { sortDate, sortValueByLength } from 'sorters/sorters';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import entityTypes from 'constants/entityTypes';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';
import { IMAGE_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';
import removeEntityContextColumns from 'utils/tableUtils';

export const defaultImageSort = [
    {
        id: 'priority',
        desc: false
    },
    {
        id: 'name.fullName',
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
            className: `w-1/6 ${defaultColumnClassName}`,
            accessor: 'name.fullName'
        },
        {
            Header: `CVEs`,
            entityType: entityTypes.CVE,
            headerClassName: `w-1/4 lg:w-1/5 xl:w-1/6 ${defaultHeaderClassName}`,
            className: `w-1/4 lg:w-1/5 xl:w-1/6 ${defaultColumnClassName}`,
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
            accessor: 'vulnCounter.all.total'
        },
        {
            Header: `Top CVSS`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { topVuln } = original;
                if (!topVuln) return '-';
                const { cvss, scoreVersion } = topVuln;
                return <TopCvssLabel cvss={cvss} version={scoreVersion} />;
            },
            accessor: 'topVuln.cvss'
        },
        {
            Header: `Created`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { metadata } = original;
                if (!metadata || !metadata.v1) return '-';
                return <DateTimeField date={metadata.v1.created} />;
            },
            sortMethod: sortDate,
            accessor: 'metadata.v1.created'
        },
        {
            Header: `Scan time`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { scan } = original;
                if (!scan) return '-';
                return <DateTimeField date={scan.scanTime} />;
            },
            sortMethod: sortDate,
            accessor: 'scan.scanTime'
        },
        {
            Header: 'Image Status',
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { deploymentCount } = original;
                const imageStatus = deploymentCount === 0 ? 'inactive' : 'active';
                return <StatusChip status={imageStatus} />;
            },
            accessor: 'deploymentCount'
        },
        {
            Header: `Deployments`,
            entityType: entityTypes.DEPLOYMENT,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { deploymentCount, id } = original;
                const url = workflowState
                    .pushListItem(id)
                    .pushList(entityTypes.DEPLOYMENT)
                    .toUrl();
                const text = `${deploymentCount} ${pluralize('deployment', deploymentCount)}`;
                return <TableCellLink pdf={pdf} url={url} text={text} />;
            },
            accessor: 'deploymentCount'
        },
        {
            Header: `Components`,
            entityType: entityTypes.COMPONENT,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { scan, id } = original;
                if (!scan) return '-';
                const { components } = scan;
                const url = workflowState
                    .pushListItem(id)
                    .pushList(entityTypes.COMPONENT)
                    .toUrl();
                const text = `${components.length} ${pluralize('component', components.length)}`;
                return <TableCellLink pdf={pdf} url={url} text={text} />;
            },
            accessor: 'scan.components',
            sortMethod: sortValueByLength
        },
        {
            Header: `Risk Priority`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            accessor: 'priority'
        }
    ];
    return removeEntityContextColumns(tableColumns, workflowState);
}

const VulnMgmtImages = ({ selectedRowId, search, sort, page, data }) => {
    const query = gql`
        query getImages($query: String) {
            results: images(query: $query) {
                ...imageFields
            }
        }
        ${IMAGE_LIST_FRAGMENT}
    `;

    const queryOptions = {
        variables: {
            query: queryService.objectToWhereClause(search)
        }
    };

    return (
        <WorkflowListPage
            data={data}
            query={query}
            queryOptions={queryOptions}
            entityListType={entityTypes.IMAGE}
            getTableColumns={getImageTableColumns}
            selectedRowId={selectedRowId}
            search={search}
            page={page}
            defaultSorted={sort}
        />
    );
};

VulnMgmtImages.propTypes = workflowListPropTypes;
VulnMgmtImages.defaultProps = {
    ...workflowListDefaultProps,
    sort: defaultImageSort
};

export default VulnMgmtImages;
