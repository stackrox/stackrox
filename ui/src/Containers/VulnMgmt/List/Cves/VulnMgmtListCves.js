/* eslint-disable react/jsx-no-bind */
import React from 'react';
import pluralize from 'pluralize';
import gql from 'graphql-tag';

import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import DateTimeField from 'Components/DateTimeField';
import LabelChip from 'Components/LabelChip';
import TableCellLink from 'Components/TableCellLink';
import TopCvssLabel from 'Components/TopCvssLabel';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';
import entityTypes from 'constants/entityTypes';
import queryService from 'modules/queryService';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';
import removeEntityContextColumns from 'utils/tableUtils';

import { VULN_CVE_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';

export function getCveTableColumns(workflowState, linksOn = true) {
    const { entityType } = workflowState.getBaseEntity();
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
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
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
        entityType !== entityTypes.IMAGE &&
            entityType !== entityTypes.COMPONENT && {
                Header: `Deployments`,
                entityType: entityTypes.DEPLOYMENT,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                    const { deploymentCount, cve } = original;
                    if (deploymentCount === 0) return 'No deployments';
                    const text = `${deploymentCount} ${pluralize('deployment', deploymentCount)}`;
                    if (!linksOn) return text;
                    const url = workflowState
                        .pushListItem(cve)
                        .pushList(entityTypes.IMAGE)
                        .toUrl();
                    return <TableCellLink pdf={pdf} url={url} text={text} />;
                },
                accessor: 'deploymentCount',
                id: 'deploymentCount'
            },
        entityType !== entityTypes.IMAGE &&
            entityType !== entityTypes.COMPONENT && {
                Header: `Images`,
                entityType: entityTypes.IMAGE,
                headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                    const { imageCount, cve } = original;
                    if (imageCount === 0) return 'No images';
                    const text = `${imageCount} ${pluralize('image', imageCount)}`;
                    if (!linksOn) return text;
                    const url = workflowState
                        .pushListItem(cve)
                        .pushList(entityTypes.IMAGE)
                        .toUrl();
                    return <TableCellLink pdf={pdf} url={url} text={text} />;
                },
                accessor: 'imageCount',
                id: 'imageCount'
            },
        entityType !== entityTypes.IMAGE &&
            entityType !== entityTypes.COMPONENT && {
                Header: `Components`,
                entityType: entityTypes.COMPONENT,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                    const { componentCount, cve } = original;
                    if (componentCount === 0) return 'No components';
                    const text = `${componentCount} ${pluralize('component', componentCount)}`;
                    if (!linksOn) return text;
                    const url = workflowState
                        .pushListItem(cve)
                        .pushList(entityTypes.IMAGE)
                        .toUrl();
                    return <TableCellLink pdf={pdf} url={url} text={text} />;
                },
                accessor: 'componentCount',
                id: 'componentCount'
            }
    ];

    const cols = removeEntityContextColumns(tableColumns, workflowState);
    // TODO: What is this about??
    // If the base page is a component or image list, then get rid of the component and image columns
    // This is a different type of logic than found on other pages. We need to erify business rules for this.
    return [entityTypes.COMPONENT, entityTypes.IMAGE].includes(entityType)
        ? cols.filter(col => ![entityTypes.COMPONENT, entityTypes.IMAGE].includes(col.entityType))
        : cols;
}

export function renderCveDescription(row) {
    const { original } = row;
    return (
        <div className="px-2 pb-4 pt-1 text-base-500">
            {original.summary || 'No description available.'}
        </div>
    );
}

export const defaultCveSort = [
    {
        id: 'lastScanned',
        desc: true
    }
];

const VulnMgmtCves = ({ selectedRowId, search, sort, page, data }) => {
    // TODO: change query line to `query getCves($query: String) {`
    //   after API starts accepting empty string ('') for query
    const CVES_QUERY = gql`
        query getCves {
            results: vulnerabilities {
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
            sort={sort}
            page={page}
            defaultSorted={defaultCveSort}
            showSubrows
            SubComponent={renderCveDescription}
        />
    );
};

VulnMgmtCves.propTypes = workflowListPropTypes;
VulnMgmtCves.defaultProps = workflowListDefaultProps;

export default VulnMgmtCves;
