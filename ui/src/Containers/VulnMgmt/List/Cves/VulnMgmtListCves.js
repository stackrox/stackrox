import React, { useContext } from 'react';
import { PropTypes } from 'prop-types';
import pluralize from 'pluralize';
import gql from 'graphql-tag';

import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import DateTimeField from 'Components/DateTimeField';
import LabelChip from 'Components/LabelChip';
import TableCellLink from 'Components/TableCellLink';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';
import entityTypes from 'constants/entityTypes';
import WorkflowStateMgr from 'modules/WorkflowStateManager';
import workflowStateContext from 'Containers/workflowStateContext';
import queryService from 'modules/queryService';
import { generateURL } from 'modules/URLReadWrite';
import { getSeverityChipType } from 'utils/vulnerabilityUtils';

import { CVE_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';

const VulnMgmtCves = ({ selectedRowId, search }) => {
    const workflowState = useContext(workflowStateContext);

    // TODO: change query line to `query getCves($query: String) {`
    //   after API starts accepting empty string ('') for query
    const CVES_QUERY = gql`
        query getCves {
            results: vulnerabilities {
                ...cveListFields
            }
        }
        ${CVE_LIST_FRAGMENT}
    `;

    const queryOptions = {
        variables: {
            query: queryService.objectToWhereClause(search)
        }
    };

    function getTableColumns() {
        const tableColumns = [
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
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                // eslint-disable-next-line
                Cell: ({ original }) => {
                    const { isFixable } = original;
                    return isFixable ? <LabelChip text="Fixable" type="success" /> : 'No';
                },
                accessor: 'isFixable',
                id: 'isFixable'
            },
            {
                Header: `CVSS score`,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                // eslint-disable-next-line
                Cell: ({ original }) => {
                    const { cvss } = original;

                    if (!cvss && cvss !== 0) return 'N/A';

                    // TODO: add CVSS version beneath when available from API
                    const chipType = getSeverityChipType(cvss);
                    return <LabelChip text={cvss.toFixed(1) || ''} type={chipType} />;
                },
                accessor: 'cvss',
                id: 'cvss'
            },
            // TODO: enable this column after data is available from the API
            // {
            //     Header: `Env. Impact`,
            //     headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            //     className: `w-1/8 ${defaultColumnClassName}`,
            //     accessor: 'envImpact'
            // },
            // TODO: enable this column after data is available from the API
            // {
            //     Header: `Impact score`,
            //     headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            //     className: `w-1/8 ${defaultColumnClassName}`,
            //     accessor: 'impactScore'
            // },
            {
                Header: `Scanned`,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                Cell: ({ original }) => {
                    const { lastScanned } = original;
                    return <DateTimeField date={lastScanned} />;
                },
                accessor: 'lastScanned',
                id: 'lastScanned'
            },
            // TODO: enable this column after data is available from the API
            // {
            //     Header: `Published`,
            //     headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            //     className: `w-1/8 ${defaultColumnClassName}`,
            //     Cell: ({ original }) => {
            //         const { published } = original;
            //         return <DateTimeField text={published} />;
            //     },
            //     id: 'published'
            // },
            {
                Header: `Deployments`,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                // eslint-disable-next-line
                Cell: ({ original, pdf }) => {
                    const { deploymentCount, cve } = original;
                    if (deploymentCount === 0) return 'No deployments';
                    const workflowStateMgr = new WorkflowStateMgr(workflowState);
                    workflowStateMgr.pushListItem(cve).pushList(entityTypes.IMAGE);
                    const url = generateURL(workflowStateMgr.workflowState);
                    return (
                        <TableCellLink
                            pdf={pdf}
                            url={url}
                            text={`${deploymentCount} ${pluralize('deployment', deploymentCount)}`}
                        />
                    );
                },
                accessor: 'deploymentCount',
                id: 'deploymentCount'
            },
            {
                Header: `Images`,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                // eslint-disable-next-line
                Cell: ({ original, pdf }) => {
                    const { imageCount, cve } = original;
                    if (imageCount === 0) return 'No images';
                    const workflowStateMgr = new WorkflowStateMgr(workflowState);
                    workflowStateMgr.pushListItem(cve).pushList(entityTypes.IMAGE);
                    const url = generateURL(workflowStateMgr.workflowState);
                    return (
                        <TableCellLink
                            pdf={pdf}
                            url={url}
                            text={`${imageCount} ${pluralize('image', imageCount)}`}
                        />
                    );
                },
                accessor: 'imageCount',
                id: 'imageCount'
            },
            {
                Header: `Components`,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                // eslint-disable-next-line
                Cell: ({ original, pdf }) => {
                    const { componentCount, cve } = original;
                    if (componentCount === 0) return 'No components';
                    const workflowStateMgr = new WorkflowStateMgr(workflowState);
                    workflowStateMgr.pushListItem(cve).pushList(entityTypes.IMAGE);
                    const url = generateURL(workflowStateMgr.workflowState);
                    return (
                        <TableCellLink
                            pdf={pdf}
                            url={url}
                            text={`${componentCount} ${pluralize('component', componentCount)}`}
                        />
                    );
                },
                accessor: 'componentCount',
                id: 'componentCount'
            }
        ];

        return tableColumns.filter(col => col); // filter out columns that are nulled based on context
    }

    return (
        <WorkflowListPage
            query={CVES_QUERY}
            queryOptions={queryOptions}
            idAttribute="cve"
            entityListType={entityTypes.CVE}
            getTableColumns={getTableColumns}
            selectedRowId={selectedRowId}
            search={search}
            defaultSorted={[
                {
                    id: 'cvss',
                    desc: true
                }
            ]}
        />
    );
};

VulnMgmtCves.propTypes = {
    selectedRowId: PropTypes.string,
    search: PropTypes.shape({})
};

VulnMgmtCves.defaultProps = {
    search: null,
    selectedRowId: null
};

export default VulnMgmtCves;
