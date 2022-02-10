import React, { ReactElement } from 'react';
import { useTable, useSortBy, useGroupBy, useExpanded, useRowSelect } from 'react-table';

import { networkFlowStatus } from 'constants/networkGraph';
import {
    networkEntityLabels,
    networkProtocolLabels,
    networkConnectionLabels,
} from 'messages/network';
import { FlattenedNetworkBaseline, BaselineStatus } from 'Containers/Network/networkTypes';

import NavigateToEntityButton from 'Containers/Network/NavigateToEntityButton';
import {
    Table,
    TableHead,
    TableBody,
    TableRow,
    TableCell,
    checkboxSelectionPlugin,
    expanderPlugin,
    TableColorStyles,
} from 'Components/TableV7';
import {
    getAggregateText,
    getDirectionalityLabel,
} from 'Containers/Network/SidePanel/NetworkDeploymentOverlay/utils';
import GroupedStatusTableCell from './GroupedStatusTableCell';
import ToggleSelectedBaselineStatuses from './ToggleSelectedBaselineStatuses';
import ToggleBaselineStatus from './ToggleBaselineStatus';
import EmptyGroupedStatusRow from './EmptyGroupedStatusRow';
import { Row } from './tableTypes';
import { getFlowRowColors } from '../networkBaseline.utils';

export type NetworkBaselinesTableProps = {
    networkBaselines: FlattenedNetworkBaseline[];
    toggleBaselineStatuses: (networkBaselines: FlattenedNetworkBaseline[]) => void;
    onNavigateToEntity: () => void;
    includedBaselineStatuses: BaselineStatus[];
    excludedColumns: string[];
};

function getEmptyGroupRow(status: BaselineStatus): Row {
    return {
        id: `status:${status}`,
        // TODO: see if we can remove this fake "peer" while keeping type-checking elsewhere
        original: {
            peer: {
                entity: {
                    id: '',
                    type: 'DEPLOYMENT', // placeholder
                    name: 'empty-group', // placeholder
                    namespace: '',
                },
                port: '',
                protocol: 'L4_PROTOCOL_ANY', // placeholder
                ingress: false,
                state: 'active', // placeholder
            },
            status,
        },
        isGrouped: true,
        groupByID: 'status',
        groupByVal: status,
        values: {
            status,
        },
        subRows: [],
        leafRows: [],
    };
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
            return getDirectionalityLabel(datum.peer.ingress);
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
            return datum.peer.entity.namespace || '-';
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
            return datum.peer.state ? networkConnectionLabels[datum.peer.state] : '-';
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
    includedBaselineStatuses,
    excludedColumns = [],
}: NetworkBaselinesTableProps): ReactElement {
    const includedColumns = React.useMemo(() => {
        if (!excludedColumns.length) {
            return columns;
        }
        const result = columns.filter(({ Header }) => !excludedColumns.includes(Header));
        return result;
    }, [excludedColumns]);
    const { headerGroups, rows, prepareRow, selectedFlatRows } = useTable(
        {
            columns: includedColumns,
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
                    [`status:${networkFlowStatus.ANOMALOUS}`]: true,
                    [`status:${networkFlowStatus.BASELINE}`]: true,
                    [`status:${networkFlowStatus.BLOCKED}`]: true,
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

    if (
        includedBaselineStatuses.includes(networkFlowStatus.ANOMALOUS as BaselineStatus) &&
        !rows.some((row: { id: string }) => row.id.includes(networkFlowStatus.ANOMALOUS))
    ) {
        const emptyAnomalousRow = getEmptyGroupRow(networkFlowStatus.ANOMALOUS as BaselineStatus);

        rows.unshift(emptyAnomalousRow);
    }

    if (
        includedBaselineStatuses.includes(networkFlowStatus.BLOCKED as BaselineStatus) &&
        !rows.some((row: { id: string }) => row.id.includes(networkFlowStatus.BLOCKED))
    ) {
        const emptyBlockedRow = getEmptyGroupRow(networkFlowStatus.BLOCKED as BaselineStatus);

        rows.unshift(emptyBlockedRow);
    }

    if (
        includedBaselineStatuses.includes(networkFlowStatus.BASELINE as BaselineStatus) &&
        !rows.some((row: { id: string }) => row.id.includes(networkFlowStatus.BASELINE))
    ) {
        const emptyBaselineRow = getEmptyGroupRow(networkFlowStatus.BASELINE as BaselineStatus);

        rows.push(emptyBaselineRow);
    }

    return (
        <div className="flex flex-1 flex-col overflow-y-auto">
            <Table>
                <TableHead headerGroups={headerGroups} />
                <TableBody>
                    {rows.map((row) => {
                        prepareRow(row);

                        const { key } = row.getRowProps();

                        // If the row is the grouped row, use the value of the group to determine it's color;
                        // otherwise, use its individual row status
                        const rowStatus =
                            row.isGrouped &&
                            (row.groupByVal === networkFlowStatus.ANOMALOUS ||
                                row.groupByVal === networkFlowStatus.BLOCKED)
                                ? row.groupByVal
                                : row.values.status;
                        const rowColorStyles: TableColorStyles = getFlowRowColors(rowStatus);

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

                        const HoveredRowComponent = row?.original?.peer?.entity?.id ? (
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
                            <React.Fragment key={key}>
                                <TableRow
                                    row={row}
                                    colorStyles={rowColorStyles}
                                    HoveredRowComponent={HoveredRowComponent}
                                    HoveredGroupedRowComponent={HoveredGroupedRowComponent}
                                    GroupedRowComponent={GroupedRowComponent}
                                >
                                    {row.isGrouped && row.groupByID === 'status' ? (
                                        <GroupedStatusTableCell
                                            colorStyles={rowColorStyles}
                                            row={row}
                                        />
                                    ) : (
                                        row.cells.map((cell) => {
                                            return (
                                                <TableCell
                                                    key={cell.column.id}
                                                    cell={cell}
                                                    colorStyles={rowColorStyles}
                                                />
                                            );
                                        })
                                    )}
                                </TableRow>
                                {row.isGrouped &&
                                    row.groupByID === 'status' &&
                                    !row.subRows.length &&
                                    !row.leafRows.length && (
                                        <EmptyGroupedStatusRow
                                            baselineStatus={row.groupByVal}
                                            columnCount={includedColumns.length}
                                        />
                                    )}
                            </React.Fragment>
                        );
                    })}
                </TableBody>
            </Table>
        </div>
    );
}

export default NetworkBaselinesTable;
