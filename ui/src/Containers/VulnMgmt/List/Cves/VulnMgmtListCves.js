/* eslint-disable react/jsx-no-bind */
import React from 'react';
import gql from 'graphql-tag';

import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import DateTimeField from 'Components/DateTimeField';
import LabelChip from 'Components/LabelChip';
import TableCountLink from 'Components/workflow/TableCountLink';
import TopCvssLabel from 'Components/TopCvssLabel';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';
import entityTypes from 'constants/entityTypes';
import queryService from 'modules/queryService';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';
import removeEntityContextColumns from 'utils/tableUtils';
import { truncate } from 'utils/textUtils';

import { VULN_CVE_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';

export const defaultCveSort = [
    {
        id: 'cvss',
        desc: true
    },
    {
        id: 'cve',
        desc: false
    }
];

export function getCveTableColumns(workflowState) {
    // to determine whether to show the counts as links in the table when not in pure CVE state
    const inFindingsSection = workflowState.getCurrentEntity().entityType !== entityTypes.CVE;
    const tableColumns = [
        {
            expander: true,
            show: false
        },
        {
            Header: 'cve',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'cve'
        },
        {
            Header: `CVE`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            accessor: 'cve'
        },
        {
            Header: `Fixable`,
            headerClassName: `w-20 text-center ${defaultHeaderClassName}`,
            className: `w-20 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original }) => {
                const { isFixable } = original;
                const fixableFlag = isFixable ? (
                    <LabelChip text="Fixable" type="success" size="large" />
                ) : (
                    'No'
                );
                return <div className="mx-auto">{fixableFlag}</div>;
            },
            accessor: 'isFixable',
            id: 'isFixable'
        },
        {
            Header: `CVSS score`,
            headerClassName: `w-1/10 text-center ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original }) => {
                const { cvss, scoreVersion } = original;
                return <TopCvssLabel cvss={cvss} version={scoreVersion} />;
            },
            accessor: 'cvss',
            id: 'cvss'
        },
        {
            Header: `Env. Impact`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { envImpact } = original;
                // eslint-disable-next-line eqeqeq
                return envImpact == Number(envImpact)
                    ? `${(envImpact * 100).toFixed(0)}% affected`
                    : '-';
            },
            accessor: 'envImpact'
        },
        {
            Header: `Impact score`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { impactScore } = original;
                // eslint-disable-next-line eqeqeq
                return impactScore == Number(impactScore) ? impactScore.toFixed(1) : '-';
            },
            accessor: 'impactScore'
        },
        {
            Header: `Scanned`,
            headerClassName: `w-1/10 text-left ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { lastScanned } = original;
                return <DateTimeField date={lastScanned} />;
            },
            accessor: 'lastScanned',
            id: 'lastScanned'
        },
        {
            Header: `Published`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { publishedOn } = original;
                return <DateTimeField date={publishedOn} />;
            },
            accessor: 'publishedOn',
            id: 'published'
        },
        {
            Header: `Deployments`,
            entityType: entityTypes.DEPLOYMENT,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => (
                <TableCountLink
                    entityType={entityTypes.DEPLOYMENT}
                    count={original.deploymentCount}
                    textOnly={inFindingsSection || pdf}
                    selectedRowId={original.cve}
                />
            ),
            accessor: 'deploymentCount',
            id: 'deploymentCount'
        },
        {
            Header: `Images`,
            entityType: entityTypes.IMAGE,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => (
                <TableCountLink
                    entityType={entityTypes.IMAGE}
                    count={original.imageCount}
                    textOnly={inFindingsSection || pdf}
                    selectedRowId={original.cve}
                />
            ),
            accessor: 'imageCount',
            id: 'imageCount'
        },
        {
            Header: `Components`,
            entityType: entityTypes.COMPONENT,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => (
                <TableCountLink
                    entityType={entityTypes.COMPONENT}
                    count={original.componentCount}
                    textOnly={inFindingsSection || pdf}
                    selectedRowId={original.cve}
                />
            ),
            accessor: 'componentCount',
            id: 'componentCount'
        }
    ];

    return removeEntityContextColumns(tableColumns, workflowState);
}

const maxLengthForSummary = 360; // based on showing up to approximately 2 lines before table starts scrolling horizontally

export function renderCveDescription(row) {
    const { original } = row;
    const truncatedSummary = truncate(original.summary, maxLengthForSummary);
    return (
        <div className="hover:bg-base-100 px-2 pb-4 pt-1 text-base-500">
            {truncatedSummary || 'No description available.'}
        </div>
    );
}

const VulnMgmtCves = ({ selectedRowId, search, sort, page, data }) => {
    // TODO: change query line to `query getCves($query: String) {`
    //   after API starts accepting empty string ('') for query
    const CVES_QUERY = gql`
        query getCves($query: String) {
            results: vulnerabilities(query: $query) {
                ...cveFields
            }
        }
        ${VULN_CVE_LIST_FRAGMENT}
    `;

    const queryOptions = {
        variables: {
            query: queryService.objectToWhereClause(search)
        }
    };

    return (
        <WorkflowListPage
            data={data}
            query={CVES_QUERY}
            queryOptions={queryOptions}
            idAttribute="cve"
            entityListType={entityTypes.CVE}
            getTableColumns={getCveTableColumns}
            selectedRowId={selectedRowId}
            search={search}
            page={page}
            defaultSorted={sort || defaultCveSort}
            showSubrows
            SubComponent={renderCveDescription}
        />
    );
};

VulnMgmtCves.propTypes = workflowListPropTypes;
VulnMgmtCves.defaultProps = {
    ...workflowListDefaultProps,
    sort: null
};

export default VulnMgmtCves;
