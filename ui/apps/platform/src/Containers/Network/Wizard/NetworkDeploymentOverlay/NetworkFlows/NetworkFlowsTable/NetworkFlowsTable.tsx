import React, { ReactElement } from 'react';
import { useTable, useSortBy, useGroupBy, useExpanded } from 'react-table';

import { networkFlowStatus } from 'constants/networkGraph';
import {
    networkEntityLabels,
    networkProtocolLabels,
    networkConnectionLabels,
} from 'messages/network';

import { CondensedButton, CondensedAlertButton } from '@stackrox/ui-components';
import Table from './Table';
import TableHead from './TableHead';
import TableBody from './TableBody';
import TableRow from './TableRow';
import TableCell from './TableCell';
import GroupedStatusTableCell from './GroupedStatusTableCell';

type NetworkFlow = {
    peer: {
        entity: {
            id: string;
            type: 'DEPLOYMENT' | 'INTERNET' | 'EXTERNAL_SOURCE';
            name: string;
            namespace?: string;
        };
        port: string;
        protocol: 'L4_PROTOCOL_TCP' | 'L4_PROTOCOL_UDP' | 'L4_PROTOCOL_ANY';
        ingress: boolean;
        state: 'active' | 'allowed';
    };
    status: 'BASELINE' | 'ANOMALOUS';
};

type Row = {
    original: NetworkFlow;
    values: {
        status: NetworkFlow['status'];
    };
};

export type NetworkFlowsTableProps = {
    networkFlows: NetworkFlow[];
};

const columns = [
    {
        Header: 'Status',
        id: 'status',
        accessor: 'status',
    },
    {
        Header: 'Entity',
        id: 'entity',
        accessor: 'peer.entity.name',
    },
    {
        Header: 'Traffic',
        id: 'traffic',
        accessor: (datum: NetworkFlow): string => {
            return datum.peer.ingress ? 'Ingress' : 'Egress';
        },
    },
    {
        Header: 'Type',
        id: 'type',
        accessor: (datum: NetworkFlow): string => {
            return networkEntityLabels[datum.peer.entity.type];
        },
    },
    {
        Header: 'Namespace',
        id: 'namespace',
        accessor: (datum: NetworkFlow): string => {
            return datum.peer.entity.namespace ?? '-';
        },
    },
    {
        Header: 'Port',
        id: 'port',
        accessor: 'peer.port',
    },
    {
        Header: 'Protocol',
        id: 'protocol',
        accessor: (datum: NetworkFlow): string => {
            return networkProtocolLabels[datum.peer.protocol];
        },
    },
    {
        Header: 'State',
        id: 'state',
        accessor: (datum: NetworkFlow): string => {
            return networkConnectionLabels[datum.peer.state];
        },
    },
];

function onHoveredRowComponentRender(row: Row): ReactElement {
    function onClick(): void {
        // TODO: remove this console log and add a way to use the API call
        // for marking as anomalous or adding to baseline
        // eslint-disable-next-line no-console
        console.log(row.original);
    }

    if (row.original.status === networkFlowStatus.ANOMALOUS) {
        return (
            <CondensedButton type="button" onClick={onClick}>
                Add to Baseline
            </CondensedButton>
        );
    }
    return (
        <CondensedAlertButton type="button" onClick={onClick}>
            Mark as Anomalous
        </CondensedAlertButton>
    );
}

function NetworkFlowsTable({ networkFlows }: NetworkFlowsTableProps): ReactElement {
    const { headerGroups, rows, prepareRow } = useTable(
        {
            columns,
            data: networkFlows,
            initialState: {
                sortBy: [
                    {
                        id: 'status',
                        desc: false,
                    },
                ],
                groupBy: ['status'],
                expanded: {
                    'status:ANOMALOUS': true,
                    'status:BASELINE': true,
                },
                hiddenColumns: ['status'],
            },
        },
        useGroupBy,
        useSortBy,
        useExpanded
    );

    return (
        <Table>
            <TableHead headerGroups={headerGroups} />
            <TableBody>
                {rows.map((row) => {
                    prepareRow(row);

                    // If the row is the grouped row or a sub row grouped by the ANOMALOUS status,
                    // we want a colored background
                    const rowType =
                        (row.isGrouped && row.groupByVal === networkFlowStatus.ANOMALOUS) ||
                        row.values.status === networkFlowStatus.ANOMALOUS
                            ? 'alert'
                            : null;

                    return (
                        <TableRow
                            key={row.id}
                            row={row}
                            type={rowType}
                            onHoveredRowComponentRender={onHoveredRowComponentRender}
                        >
                            {row.isGrouped ? (
                                <GroupedStatusTableCell row={row} />
                            ) : (
                                row.cells.map((cell) => {
                                    return <TableCell key={cell.column.Header} cell={cell} />;
                                })
                            )}
                        </TableRow>
                    );
                })}
            </TableBody>
        </Table>
    );
}

export default NetworkFlowsTable;
