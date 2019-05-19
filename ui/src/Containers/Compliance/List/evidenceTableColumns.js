import React from 'react';
import ComplianceStateLabel from 'Containers/Compliance/ComplianceStateLabel';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import { standardLabels } from 'messages/standards';
import { sortValue } from 'sorters/sorters';
import entityTypes from 'constants/entityTypes';
import upperCase from 'lodash/upperCase';

const tableColumnData = {
    controlId: {
        Header: 'id',
        headerClassName: 'hidden',
        className: 'hidden',
        accessor: 'control.id'
    },
    standardId: {
        Header: `Standard`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'control.standardId',
        Cell: ({ original }) => standardLabels[original.control.standardId]
    },
    controlName: {
        Header: `Control`,
        headerClassName: `w-1/4 ${defaultHeaderClassName}`,
        className: `w-1/4 ${defaultColumnClassName}`,
        accessor: 'control.name',
        sortMethod: sortValue,
        Cell: ({ original }) => `${original.control.name} - ${original.control.description}`
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
    namespaceName: {
        Header: `Namespace`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'resource.namespace',
        Cell: ({ original }) => original.resource.namespace || '-'
    },
    entityName: {
        Header: `Entity`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'resource.name'
    },
    resourceType: {
        Header: `Type`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'resource.__typename'
    },
    clusterName: {
        Header: `Cluster`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'resource.clusterName',
        Cell: ({ original }) => {
            // eslint-disable-next-line
            if (upperCase(original.resource.__typename) === entityTypes.CLUSTER) {
                return original.resource.name || '-';
            }
            return original.resource.clusterName || '-';
        }
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

export const controlsTableColumns = [
    tableColumnData.controlId,
    tableColumnData.standardId,
    tableColumnData.controlName,
    tableColumnData.state,
    tableColumnData.entityName,
    tableColumnData.resourceType,
    tableColumnData.namespaceName,
    tableColumnData.clusterName,
    tableColumnData.evidence
];

export const clustersTableColumns = [
    tableColumnData.resourceId,
    tableColumnData.clusterName,
    tableColumnData.state,
    tableColumnData.evidence
];

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
