import React from 'react';
import ComplianceStateLabel from 'Containers/Compliance/ComplianceStateLabel';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';

const tableColumnData = {
    controlId: {
        Header: 'id',
        headerClassName: 'hidden',
        className: 'hidden',
        accessor: 'control.id'
    },
    resourceId: {
        Header: `id`,
        headerClassName: 'hidden',
        className: 'hidden',
        accessor: 'resource.id'
    },
    deploymentName: {
        Header: `Deployment`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'resource.name'
    },
    nodeName: {
        Header: `Node`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'resource.name'
    },
    clusterName: {
        Header: `Cluster`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'resource.clusterName'
    },
    state: {
        Header: `State`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'value.overallState',
        // eslint-disable-next-line
        Cell: ({ original }) => <ComplianceStateLabel state={original.value.overallState} />
    },
    evidence: {
        Header: `Evidence`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'value.evidence',
        // eslint-disable-next-line
        Cell: ({ original }) => {
            const { length } = original.value.evidence;
            return length > 1 ? (
                <div className="italic font-800">{`Inspect to view ${length} pieces of evidence`}</div>
            ) : (
                original.value.evidence[0].message
            );
        }
    }
};

export const nodesTableColumns = [
    tableColumnData.resourceId,
    tableColumnData.nodeName,
    tableColumnData.clusterName,
    tableColumnData.state,
    tableColumnData.evidence
];

export const deploymentsTableColumns = [
    tableColumnData.resourceId,
    tableColumnData.deploymentName,
    tableColumnData.clusterName,
    tableColumnData.state,
    tableColumnData.evidence
];
