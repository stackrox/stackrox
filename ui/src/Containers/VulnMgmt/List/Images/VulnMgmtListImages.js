import React from 'react';
import { PropTypes } from 'prop-types';
import gql from 'graphql-tag';
import pluralize from 'pluralize';

import queryService from 'modules/queryService';
import TopCvssLabel from 'Components/TopCvssLabel';
import TableCellLink from 'Components/TableCellLink';
// import FixableCVECount from 'Components/FixableCVECount';
// import SeverityStackedPill from 'Components/visuals/SeverityStackedPill';
import DateTimeField from 'Components/DateTimeField';
import { sortDate } from 'sorters/sorters';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import entityTypes from 'constants/entityTypes';
import WorkflowStateMgr from 'modules/WorkflowStateManager';
import { generateURL } from 'modules/URLReadWrite';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';
import { IMAGE_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';

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
        // TO DO: add back once vulnCounter goes in
        // {
        //     Header: `CVEs`,
        //     headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        //     className: `w-1/8 ${defaultColumnClassName}`,
        //     Cell: ({ original, pdf }) => {
        //         const { vulnCounter, id } = original;
        //         const workflowStateMgr = new WorkflowStateMgr(workflowState);
        //         workflowStateMgr.pushListItem(id).pushList(entityTypes.CVE);
        //         const url = generateURL(workflowStateMgr.workflowState);
        //         const { critical, high, medium, low, fixable, total } = vulnCounter;
        //         return (
        //             <div className="flex w-full items-center">
        //                 <FixableCVECount
        //                     cves={total}
        //                     fixable={fixable}
        //                     orientation="vertical"
        //                     url={url}
        //                     pdf={pdf}
        //                 />
        //                 <SeverityStackedPill
        //                     critical={critical}
        //                     high={high}
        //                     medium={medium}
        //                     low={low}
        //                 />
        //             </div>
        //         );
        //     }
        // },
        {
            Header: `Top CVSS`,
            headerClassName: `w-1 ${defaultHeaderClassName}`,
            className: `w-1 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { topVuln } = original;
                const { cvss, scoreVersion } = topVuln;
                return <TopCvssLabel cvss={cvss} version={scoreVersion} />;
            }
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
            sortMethod: sortDate
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
            sortMethod: sortDate
        },
        // TO DO: add image status column once backend is ready
        {
            Header: `Deployments`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { deploymentCount, id } = original;
                const workflowStateMgr = new WorkflowStateMgr(workflowState);
                workflowStateMgr.pushListItem(id).pushList(entityTypes.DEPLOYMENT);
                const url = generateURL(workflowStateMgr.workflowState);
                const text = `${deploymentCount} ${pluralize('deployment', deploymentCount)}`;
                return <TableCellLink pdf={pdf} url={url} text={text} />;
            }
        },
        {
            Header: `Components`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { scan, id } = original;
                if (!scan) return '-';
                const { components } = scan;
                const workflowStateMgr = new WorkflowStateMgr(workflowState);
                workflowStateMgr.pushListItem(id).pushList(entityTypes.COMPONENT);
                const url = generateURL(workflowStateMgr.workflowState);
                const text = `${components.length} ${pluralize('component', components.length)}`;
                return <TableCellLink pdf={pdf} url={url} text={text} />;
            }
        },
        {
            Header: `Risk`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            accessor: 'priority',
            Cell: ({ original }) => original.priority + 1
        }
    ];
    return tableColumns.filter(col => col);
}

const VulnMgmtImages = ({ selectedRowId, search }) => {
    const query = gql`
        query getImages {
            results: images {
                ...imageListFields
            }
        }
        ${IMAGE_LIST_FRAGMENT}
    `;

    const queryOptions = {
        variables: {
            query: queryService.objectToWhereClause(search)
        }
    };

    const defaultImageSort = [
        {
            id: 'priority',
            desc: false
        }
    ];

    return (
        <WorkflowListPage
            query={query}
            queryOptions={queryOptions}
            entityListType={entityTypes.IMAGE}
            getTableColumns={getImageTableColumns}
            selectedRowId={selectedRowId}
            search={search}
            defaultSorted={defaultImageSort}
        />
    );
};

VulnMgmtImages.propTypes = {
    selectedRowId: PropTypes.string,
    search: PropTypes.shape({}),
    entityContext: PropTypes.shape({})
};

VulnMgmtImages.defaultProps = {
    search: null,
    entityContext: {},
    selectedRowId: null
};

export default VulnMgmtImages;
