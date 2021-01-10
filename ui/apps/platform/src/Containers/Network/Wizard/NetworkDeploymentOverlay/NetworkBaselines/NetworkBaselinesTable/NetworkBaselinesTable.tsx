/* eslint-disable react/display-name */
import React, { ReactElement } from 'react';
import { useTable, useSortBy, useGroupBy, useExpanded, useRowSelect } from 'react-table';
import uniq from 'lodash/uniq';

import { networkFlowStatus } from 'constants/networkGraph';
import {
    networkEntityLabels,
    networkProtocolLabels,
    networkConnectionLabels,
} from 'messages/network';
import { FlattenedNetworkBaseline } from 'Containers/Network/networkTypes';

import NavigateToEntityButton from 'Containers/Network/NavigateToEntityButton';
import Table from './Table';
import TableHead from './TableHead';
import TableBody from './TableBody';
import TableRow from './TableRow';
import TableCell from './TableCell';
import GroupedStatusTableCell from './GroupedStatusTableCell';
import ToggleSelectedBaselineStatuses from './ToggleSelectedBaselineStatuses';
import ToggleBaselineStatus from './ToggleBaselineStatus';
import checkboxSelectionPlugin from './checkboxSelectionPlugin';
import expanderPlugin from './expanderPlugin';

export type NetworkBaselinesTableProps = {
    networkBaselines: FlattenedNetworkBaseline[];
    toggleBaselineStatuses: (networkBaselines: FlattenedNetworkBaseline[]) => void;
    onNavigateToEntity: () => void;
};

function getAggregateText(leafValues: string[], multiplePhrase = 'Many'): string {
    const uniqValues = uniq(leafValues);
    if (uniqValues.length > 1) {
        return multiplePhrase;
    }
    return uniqValues[0];
}

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
        accessor: (datum: FlattenedNetworkBaseline): string => {
            if (datum.peer.ingress) {
                return 'Ingress';
            }
            return 'Egress';
        },
        aggregate: (leafValues: string[]): string => {
            return getAggregateText(leafValues, 'Two-way');
        },
    },
    {
        Header: 'Type',
        id: 'type',
        accessor: (datum: FlattenedNetworkBaseline): string => {
            return networkEntityLabels[datum.peer.entity.type];
        },
        aggregate: (leafValues: string[]): string => {
            return getAggregateText(leafValues);
        },
    },
    {
        Header: 'Namespace',
        id: 'namespace',
        accessor: (datum: FlattenedNetworkBaseline): string => {
            return datum.peer.entity.namespace ?? '-';
        },
        aggregate: (leafValues: string[]): string => {
            return getAggregateText(leafValues);
        },
    },
    {
        Header: 'Port',
        id: 'port',
        accessor: 'peer.port',
        aggregate: (leafValues: string[]): string => {
            return getAggregateText(leafValues);
        },
    },
    {
        Header: 'Protocol',
        id: 'protocol',
        accessor: (datum: FlattenedNetworkBaseline): string => {
            return networkProtocolLabels[datum.peer.protocol];
        },
        aggregate: (leafValues: string[]): string => {
            return getAggregateText(leafValues);
        },
    },
    {
        Header: 'State',
        id: 'state',
        accessor: (datum: FlattenedNetworkBaseline): string => {
            return networkConnectionLabels[datum.peer.state];
        },
        aggregate: (leafValues: string[]): string => {
            return getAggregateText(leafValues);
        },
    },
];

function NetworkBaselinesTable({
    networkBaselines,
    toggleBaselineStatuses,
    onNavigateToEntity,
}: NetworkBaselinesTableProps): ReactElement {
    const { headerGroups, rows, prepareRow, selectedFlatRows } = useTable(
        {
            columns,
            data: networkBaselines,
            initialState: {
                sortBy: [
                    {
                        id: 'status',
                        desc: false,
                    },
                ],
                groupBy: ['status', 'entity'],
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
        checkboxSelectionPlugin,
        expanderPlugin
    );

    return (
        <div className="flex flex-1 flex-col overflow-y-auto">
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

                        const GroupedRowComponent =
                            row.groupByID === 'status' ? (
                                <ToggleSelectedBaselineStatuses
                                    rows={rows}
                                    row={row}
                                    selectedFlatRows={selectedFlatRows}
                                    toggleBaselineStatuses={toggleBaselineStatuses}
                                />
                            ) : null;

                        const HoveredGroupedRowComponent =
                            row.groupByID !== 'status' && row.subRows.length >= 1 ? (
                                <div className="flex">
                                    {row.subRows.length === 1 && (
                                        <ToggleBaselineStatus
                                            row={row.subRows[0]}
                                            toggleBaselineStatuses={toggleBaselineStatuses}
                                        />
                                    )}
                                    <div className="ml-2">
                                        <NavigateToEntityButton
                                            entityId={row.subRows[0].original.peer.entity.id}
                                            entityType={row.subRows[0].original.peer.entity.type}
                                            onClick={onNavigateToEntity}
                                        />
                                    </div>
                                </div>
                            ) : null;

                        const HoveredRowComponent = row.original ? (
                            <div className="flex">
                                <ToggleBaselineStatus
                                    row={row}
                                    toggleBaselineStatuses={toggleBaselineStatuses}
                                />
                                <div className="ml-2">
                                    <NavigateToEntityButton
                                        entityId={row.original.peer.entity.id}
                                        entityType={row.original.peer.entity.type}
                                        onClick={onNavigateToEntity}
                                    />
                                </div>
                            </div>
                        ) : null;

                        return (
                            <TableRow
                                key={row.id}
                                row={row}
                                type={rowType}
                                HoveredRowComponent={HoveredRowComponent}
                                HoveredGroupedRowComponent={HoveredGroupedRowComponent}
                                GroupedRowComponent={GroupedRowComponent}
                            >
                                {row.isGrouped && row.groupByID === 'status' ? (
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
        </div>
    );
}

export default NetworkBaselinesTable;
