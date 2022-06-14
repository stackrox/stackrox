import React, { ReactElement } from 'react';
import { useTable, useSortBy, useGroupBy, useExpanded } from 'react-table';

import {
    networkEntityLabels,
    networkProtocolLabels,
    networkConnectionLabels,
} from 'messages/network';
import {
    Table,
    TableHead,
    TableBody,
    TableRow,
    TableCell,
    expanderPlugin,
} from 'Components/TableV7';
import {
    getAggregateText,
    getDirectionalityLabel,
} from 'Containers/Network/SidePanel/NetworkDeploymentOverlay/utils';
import Loader from 'Components/Loader';
import { SimulatedBaseline } from 'Containers/Network/networkTypes';
import getRowColorStylesByStatus from './getRowColorStylesByStatus';

const columns = [
    {
        Header: 'Status',
        id: 'simulatedStatus',
        accessor: 'simulatedStatus',
        hidden: true,
        aggregate: (leafValues: string[]): string => {
            return getAggregateText(leafValues);
        },
    },
    {
        Header: 'Entity',
        id: 'entity',
        accessor: 'peer.entity.name',
        aggregate: (leafValues: string[]): string => {
            return getAggregateText(leafValues);
        },
    },
    {
        Header: 'Traffic',
        id: 'traffic',
        accessor: (datum: SimulatedBaseline): string => {
            const { ingress } = datum.peer;
            return getDirectionalityLabel(ingress);
        },
        aggregate: (leafValues: string[]): string => {
            return getAggregateText(leafValues, 'Two-way');
        },
    },
    {
        Header: 'Type',
        id: 'type',
        accessor: (datum: SimulatedBaseline): string => {
            return networkEntityLabels[datum.peer.entity.type];
        },
        aggregate: (leafValues: string[]): string => {
            return getAggregateText(leafValues);
        },
    },
    {
        Header: 'Namespace',
        id: 'namespace',
        accessor: (datum: SimulatedBaseline): string => {
            // TODO: Reference https://github.com/stackrox/stackrox/pull/8005#discussion_r612485102
            return datum.peer.entity.namespace || '-';
        },
        aggregate: (leafValues: string[]): string => {
            return getAggregateText(leafValues);
        },
    },
    {
        Header: 'Port',
        id: 'port',
        accessor: (datum: SimulatedBaseline): string => {
            return datum.peer.port;
        },
        aggregate: (leafValues: string[]): string => {
            return getAggregateText(leafValues);
        },
    },
    {
        Header: 'Protocol',
        id: 'protocol',
        accessor: (datum: SimulatedBaseline): string => {
            const { protocol } = datum.peer;
            return networkProtocolLabels[protocol];
        },
        aggregate: (leafValues: string[]): string => {
            return getAggregateText(leafValues);
        },
    },
    {
        Header: 'State',
        id: 'state',
        accessor: (datum: SimulatedBaseline): string => {
            return networkConnectionLabels[datum.peer.state];
        },
        aggregate: (leafValues: string[]): string => {
            return getAggregateText(leafValues);
        },
    },
];

function SimulatedNeworkBaselines({ simulatedBaselines, isLoading }): ReactElement {
    const { headerGroups, rows, prepareRow } = useTable(
        {
            columns,
            data: simulatedBaselines,
            initialState: {
                sortBy: [
                    {
                        id: 'simulatedStatus',
                        desc: false,
                    },
                ],
                groupBy: ['entity'],
                hiddenColumns: ['simulatedStatus'],
            },
        },
        useGroupBy,
        useSortBy,
        useExpanded,
        expanderPlugin
    );

    if (isLoading) {
        return (
            <div className="p-4 w-full">
                <Loader message={null} />
            </div>
        );
    }

    return (
        <div className="flex flex-1 flex-col overflow-y-auto">
            <Table>
                <TableHead headerGroups={headerGroups} />
                <TableBody>
                    {rows.map((row) => {
                        prepareRow(row);

                        const { key } = row.getRowProps();
                        const rowColorStyles = getRowColorStylesByStatus(
                            row.values.simulatedStatus
                        );

                        return (
                            <TableRow key={key} row={row} colorStyles={rowColorStyles}>
                                {row.cells.map((cell) => {
                                    return (
                                        <TableCell
                                            key={cell.column.Header}
                                            cell={cell}
                                            colorStyles={rowColorStyles}
                                        />
                                    );
                                })}
                            </TableRow>
                        );
                    })}
                </TableBody>
            </Table>
        </div>
    );
}

export default React.memo(SimulatedNeworkBaselines);
