/* eslint-disable react/display-name */
import React, { ReactElement } from 'react';
import { useTable, useSortBy, useGroupBy, useExpanded, useRowSelect } from 'react-table';

import { networkFlowStatus } from 'constants/networkGraph';
import {
    networkEntityLabels,
    networkProtocolLabels,
    networkConnectionLabels,
} from 'messages/network';
import { CondensedButton, CondensedAlertButton } from '@stackrox/ui-components';
import { NetworkBaseline } from '../networkTypes';

import Table from './Table';
import TableHead from './TableHead';
import TableBody from './TableBody';
import TableRow from './TableRow';
import TableCell from './TableCell';
import GroupedStatusTableCell from './GroupedStatusTableCell';
import checkboxSelectionPlugin from './checkboxSelectionPlugin';

export type Row = {
    original: NetworkBaseline;
    values: {
        status: NetworkBaseline['status'];
    };
    groupByVal: NetworkBaseline['status'];
};

export type NetworkFlowsTableProps = {
    networkFlows: NetworkBaseline[];
};

export type HoveredRowComponentProps = {
    row: Row;
};

export type GroupedRowComponentProps = {
    row: Row;
    rows: Row[];
    selectedFlatRows: Row[];
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
        // eslint-disable-next-line react/display-name
        accessor: (datum: NetworkBaseline): ReactElement => {
            return <div className="ml-2">{datum.peer.entity.name}</div>;
        },
    },
    {
        Header: 'Traffic',
        id: 'traffic',
        accessor: (datum: NetworkBaseline): string => {
            if (datum.peer.ingress && datum.peer.egress) {
                return 'Bidirectional';
            }
            if (datum.peer.ingress) {
                return 'Ingress';
            }
            return 'Egress';
        },
    },
    {
        Header: 'Type',
        id: 'type',
        accessor: (datum: NetworkBaseline): string => {
            return networkEntityLabels[datum.peer.entity.type];
        },
    },
    {
        Header: 'Namespace',
        id: 'namespace',
        accessor: (datum: NetworkBaseline): string => {
            return datum.peer.entity.namespace ?? '-';
        },
    },
    {
        Header: 'Port',
        id: 'port',
        accessor: (datum: NetworkBaseline): string => {
            if (datum.peer.portsAndProtocols.length > 1) {
                return 'Multiple';
            }
            return datum.peer.portsAndProtocols[0].port;
        },
    },
    {
        Header: 'Protocol',
        id: 'protocol',
        accessor: (datum: NetworkBaseline): string => {
            if (datum.peer.portsAndProtocols.length > 1) {
                return 'Multiple';
            }
            return networkProtocolLabels[datum.peer.portsAndProtocols[0].protocol];
        },
    },
    {
        Header: 'State',
        id: 'state',
        accessor: (datum: NetworkBaseline): string => {
            return networkConnectionLabels[datum.peer.state];
        },
    },
];

// TODO: Separate into different file
function HoveredRowComponent({ row }: HoveredRowComponentProps): ReactElement {
    function onClick(): void {
        // TODO: remove this console log and add a way to use the API call
        // for marking as anomalous or adding to baseline
        // eslint-disable-next-line no-console
        console.log(row.original);
    }

    if (row.original.status === networkFlowStatus.ANOMALOUS) {
        return (
            <CondensedButton type="button" onClick={onClick}>
                Add to baseline
            </CondensedButton>
        );
    }
    return (
        <CondensedAlertButton type="button" onClick={onClick}>
            Mark as anomalous
        </CondensedAlertButton>
    );
}

// TODO: Separate into different file
function GroupedRowComponent({
    rows,
    row,
    selectedFlatRows,
}: GroupedRowComponentProps): ReactElement {
    const anomalousSelectedRows = selectedFlatRows.filter(
        (datum) => datum?.original?.status === networkFlowStatus.ANOMALOUS
    );
    const baselineSelectedRows = selectedFlatRows.filter(
        (datum) => datum?.original?.status === networkFlowStatus.BASELINE
    );
    const isAnomalousGroup = row.groupByVal === networkFlowStatus.ANOMALOUS;

    function onClick(): void {
        if (isAnomalousGroup) {
            if (anomalousSelectedRows.length) {
                // Replace this with an API call to mark selected rows as anomalous
                // eslint-disable-next-line no-console
                console.log('mark selected as anomalous', anomalousSelectedRows);
            } else {
                const allAnomalousRows = rows.filter(
                    (datum) => datum?.original?.status === networkFlowStatus.ANOMALOUS
                );
                // Replace this with an API call to mark all rows as anomalous
                // eslint-disable-next-line no-console
                console.log('mark all anomalous', allAnomalousRows);
            }
        } else if (baselineSelectedRows.length) {
            // Replace this with an API call to add selected rows to baseline
            // eslint-disable-next-line no-console
            console.log('add selected to baseline', baselineSelectedRows);
        } else {
            const allBaselineRows = rows.filter(
                (datum) => datum?.original?.status === networkFlowStatus.BASELINE
            );
            // Replace this with an API call to add all rows to baseline
            // eslint-disable-next-line no-console
            console.log('add all baseline', allBaselineRows);
        }
    }

    if (isAnomalousGroup) {
        return (
            <CondensedButton type="button" onClick={onClick}>
                Add {anomalousSelectedRows.length || 'all'} to baseline
            </CondensedButton>
        );
    }
    return (
        <CondensedAlertButton type="button" onClick={onClick}>
            Mark {baselineSelectedRows.length || 'all'} as anomalous
        </CondensedAlertButton>
    );
}

function NetworkFlowsTable({ networkFlows }: NetworkFlowsTableProps): ReactElement {
    const { headerGroups, rows, prepareRow, selectedFlatRows } = useTable(
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
        useExpanded,
        useRowSelect,
        checkboxSelectionPlugin
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
                            HoveredRowComponent={<HoveredRowComponent row={row} />}
                            GroupedRowComponent={
                                <GroupedRowComponent
                                    rows={rows}
                                    row={row}
                                    selectedFlatRows={selectedFlatRows}
                                />
                            }
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
