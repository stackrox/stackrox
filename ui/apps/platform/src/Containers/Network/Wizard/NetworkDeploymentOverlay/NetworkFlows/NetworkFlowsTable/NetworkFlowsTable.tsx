import React, { ReactElement } from 'react';
import { useTable, useSortBy } from 'react-table';

import {
    networkEntityLabels,
    networkProtocolLabels,
    networkConnectionLabels,
} from 'messages/network';

import Table from './Table';
import TableHead from './TableHead';
import TableBody from './TableBody';
import TableRow from './TableRow';
import TableCell from './TableCell';

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
                hiddenColumns: ['status'],
            },
        },
        useSortBy
    );

    return (
        <Table>
            <TableHead headerGroups={headerGroups} />
            <TableBody>
                {rows.map((row) => {
                    prepareRow(row);
                    return (
                        <TableRow key={row.id} row={row}>
                            {row.cells.map((cell) => {
                                return <TableCell key={cell.column.Header} cell={cell} />;
                            })}
                        </TableRow>
                    );
                })}
            </TableBody>
        </Table>
    );
}

export default NetworkFlowsTable;
