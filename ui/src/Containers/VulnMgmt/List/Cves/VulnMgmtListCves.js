/* eslint-disable react/jsx-no-bind */
import React, { useState } from 'react';
import gql from 'graphql-tag';
import * as Icon from 'react-feather';

import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import RowActionButton from 'Components/RowActionButton';
import DateTimeField from 'Components/DateTimeField';
import LabelChip from 'Components/LabelChip';
import TableCountLink from 'Components/workflow/TableCountLink';
import TopCvssLabel from 'Components/TopCvssLabel';
import PanelButton from 'Components/PanelButton';
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
    const [selectedCves, setSelectedCves] = useState([]);
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

    const addToPolicy = cve => e => {
        e.stopPropagation();
        if (cve) {
            // to do: add add to policy logic
        } else {
            setSelectedCves([]);
        }
    };
    const suppressCVEs = cve => e => {
        e.stopPropagation();
        if (cve) {
            // to do: add suppress logic
        } else {
            setSelectedCves([]);
        }
    };
    const viewSuppressed = () => {
        // console.log('view suppressed');
    };

    const renderRowActionButtons = ({ id }) => (
        <div className="flex border-2 border-r-2 border-base-400 bg-base-100">
            <RowActionButton
                text="Add to Policy"
                onClick={addToPolicy(id)}
                icon={<Icon.Plus className="mt-1 h-4 w-4" />}
            />
            <RowActionButton
                text="Suppress CVE"
                border="border-l-2 border-base-400"
                onClick={suppressCVEs(id)}
                icon={<Icon.Trash2 className="mt-1 h-4 w-4" />}
            />
        </div>
    );

    const tableHeaderComponents = (
        <React.Fragment>
            <PanelButton
                icon={<Icon.Plus className="h-4 w-4" />}
                className="btn-icon btn-tertiary"
                onClick={addToPolicy()}
                disabled={selectedCves.length === 0}
            >
                Add to Policy
            </PanelButton>
            <PanelButton
                icon={<Icon.Trash2 className="h-4 w-4" />}
                className="btn-icon btn-alert ml-2"
                onClick={suppressCVEs()}
                disabled={selectedCves.length === 0}
            >
                Suppress
            </PanelButton>
            <PanelButton
                icon={<Icon.Plus className="h-4 w-4" />}
                className="btn-icon btn-base ml-2"
                onClick={viewSuppressed}
            >
                View Suppressed
            </PanelButton>
        </React.Fragment>
    );

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
            checkbox
            tableHeaderComponents={tableHeaderComponents}
            selection={selectedCves}
            setSelection={setSelectedCves}
            renderRowActionButtons={renderRowActionButtons}
        />
    );
};

VulnMgmtCves.propTypes = workflowListPropTypes;
VulnMgmtCves.defaultProps = {
    ...workflowListDefaultProps,
    sort: null
};

export default VulnMgmtCves;
