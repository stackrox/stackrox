import React from 'react';
import pluralize from 'pluralize';
import gql from 'graphql-tag';

import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import TopCvssLabel from 'Components/TopCvssLabel';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';
import entityTypes from 'constants/entityTypes';
import CVEStackedPill from 'Components/CVEStackedPill';
import TableCellLink from 'Components/TableCellLink';
import queryService from 'modules/queryService';

import { COMPONENT_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';

export function getComponentTableColumns(workflowState) {
    const tableColumns = [
        {
            Header: 'Id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'id'
        },
        {
            Header: `Component`,
            headerClassName: `w-1/4 ${defaultHeaderClassName}`,
            className: `w-1/4 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { version, name } = original;
                return `${name} ${version}`;
            }
        },
        {
            Header: `CVEs`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { vulnCounter, id } = original;
                if (!vulnCounter || vulnCounter.all.total === 0) return 'No CVEs';
                const url = workflowState
                    .pushListItem(id)
                    .pushList(entityTypes.CVE)
                    .toUrl();
                return <CVEStackedPill vulnCounter={vulnCounter} url={url} pdf={pdf} />;
            }
        },
        {
            Header: `Top CVSS`,
            headerClassName: `w-1/8 text-center ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { topVuln } = original;
                const { cvss, scoreVersion } = topVuln;
                return <TopCvssLabel cvss={cvss} version={scoreVersion} />;
            }
        },
        {
            Header: `Images`,
            headerClassName: `w-1/6 ${defaultHeaderClassName}`,
            className: `w-1/6 ${defaultColumnClassName}`,
            accessor: 'imageCount',
            Cell: ({ original, pdf }) => {
                const { imageCount, id } = original;
                const url = workflowState
                    .pushListItem(id)
                    .pushList(entityTypes.IMAGE)
                    .toUrl();
                const text = `${imageCount} ${pluralize(
                    entityTypes.IMAGE.toLowerCase(),
                    imageCount
                )}`;
                return <TableCellLink pdf={pdf} url={url} text={text} />;
            }
        },
        {
            Header: `Deployments`,
            headerClassName: `w-1/6 ${defaultHeaderClassName}`,
            className: `w-1/6 ${defaultColumnClassName}`,
            accessor: 'deploymentCount',
            Cell: ({ original, pdf }) => {
                const { deploymentCount, id } = original;
                const url = workflowState
                    .pushListItem(id)
                    .pushList(entityTypes.DEPLOYMENT)
                    .toUrl();
                const text = `${deploymentCount} ${pluralize(
                    entityTypes.DEPLOYMENT.toLowerCase(),
                    deploymentCount
                )}`;
                return <TableCellLink pdf={pdf} url={url} text={text} />;
            }
        },
        {
            Header: `Risk Priority`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'priority'
        }
    ];

    return tableColumns.filter(col => col); // filter out columns that are nulled based on context
}

export const defaultComponentSort = [
    {
        id: 'priority',
        desc: false
    }
];

const VulnMgmtComponents = ({ selectedRowId, search, sort, page }) => {
    const query = gql`
        query getComponents($query: String) {
            results: imageComponents(query: $query) {
                ...componentListFields
            }
        }
        ${COMPONENT_LIST_FRAGMENT}
    `;

    const queryOptions = {
        variables: {
            query: queryService.objectToWhereClause(search)
        }
    };

    return (
        <WorkflowListPage
            query={query}
            queryOptions={queryOptions}
            idAttribute="id"
            entityListType={entityTypes.COMPONENT}
            defaultSorted={defaultComponentSort}
            getTableColumns={getComponentTableColumns}
            selectedRowId={selectedRowId}
            page={page}
            search={search}
            sort={sort}
        />
    );
};

VulnMgmtComponents.propTypes = workflowListPropTypes;
VulnMgmtComponents.defaultProps = workflowListDefaultProps;

export default VulnMgmtComponents;
