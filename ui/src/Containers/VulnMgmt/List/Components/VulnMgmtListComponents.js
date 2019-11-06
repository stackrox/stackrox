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

import { VULN_COMPONENT_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';
import removeEntityContextColumns from 'utils/tableUtils';

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
            },
            accessor: 'name'
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
            headerClassName: `w-1/8 text-center ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { topVuln } = original;
                const { cvss, scoreVersion } = topVuln;
                return <TopCvssLabel cvss={cvss} version={scoreVersion} />;
            },
            accessor: 'topVuln.cvss'
        },
        {
            Header: `Images`,
            entityType: entityTypes.IMAGE,
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
            entityType: entityTypes.DEPLOYMENT,
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

    return removeEntityContextColumns(tableColumns, workflowState);
}

export const defaultComponentSort = [
    {
        id: 'priority',
        desc: false
    }
];

const VulnMgmtComponents = ({ selectedRowId, search, sort, page, data }) => {
    const query = gql`
        query getComponents($query: String) {
            results: imageComponents(query: $query) {
                ...componentFields
            }
        }
        ${VULN_COMPONENT_LIST_FRAGMENT}
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
