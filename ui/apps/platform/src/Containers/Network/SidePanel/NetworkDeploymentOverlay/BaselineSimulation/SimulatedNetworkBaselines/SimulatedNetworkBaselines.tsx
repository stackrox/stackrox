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
import { getPropertiesByStatus, getRowColorStylesByStatus } from './utils';
import { SimulatedBaseline, ModifiedBaseline } from './baselineSimulationTypes';

type ModifiedValueProps = {
    addedValue: string;
    removedValue: string;
};

function ModifiedValue({ addedValue, removedValue }: ModifiedValueProps): ReactElement {
    return (
        <div>
            <div>{addedValue}</div>
            <s>{removedValue}</s>
        </div>
    );
}

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
        accessor: (datum: SimulatedBaseline): string | ReactElement => {
            const { simulatedStatus } = datum;
            if (simulatedStatus === 'MODIFIED') {
                const modifiedBaseline = datum as ModifiedBaseline;
                const addedValue = getDirectionalityLabel(
                    modifiedBaseline.peer.modified.added.ingress
                );
                const removedValue = getDirectionalityLabel(
                    modifiedBaseline.peer.modified.removed.ingress
                );
                return <ModifiedValue addedValue={addedValue} removedValue={removedValue} />;
            }
            const { ingress } = getPropertiesByStatus(datum);
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
            // TODO: Reference https://github.com/stackrox/rox/pull/8005#discussion_r612485102
            return datum.peer.entity.namespace || '-';
        },
        aggregate: (leafValues: string[]): string => {
            return getAggregateText(leafValues);
        },
    },
    {
        Header: 'Port',
        id: 'port',
        accessor: (datum: SimulatedBaseline): string | ReactElement => {
            const { simulatedStatus } = datum;
            if (simulatedStatus === 'MODIFIED') {
                const modifiedBaseline = datum as ModifiedBaseline;
                const addedValue = modifiedBaseline.peer.modified.added.port;
                const removedValue = modifiedBaseline.peer.modified.removed.port;
                return <ModifiedValue addedValue={addedValue} removedValue={removedValue} />;
            }
            const { port } = getPropertiesByStatus(datum);
            return port;
        },
        aggregate: (leafValues: string[]): string => {
            return getAggregateText(leafValues);
        },
    },
    {
        Header: 'Protocol',
        id: 'protocol',
        accessor: (datum: SimulatedBaseline): string | ReactElement => {
            const { simulatedStatus } = datum;
            if (simulatedStatus === 'MODIFIED') {
                const modifiedBaseline = datum as ModifiedBaseline;
                const addedValue =
                    networkProtocolLabels[modifiedBaseline.peer.modified.added.protocol];
                const removedValue =
                    networkProtocolLabels[modifiedBaseline.peer.modified.removed.protocol];
                return <ModifiedValue addedValue={addedValue} removedValue={removedValue} />;
            }
            const { protocol } = getPropertiesByStatus(datum);
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

function SimulatedNeworkBaselines({ simulatedNetworkBaselines }): ReactElement {
    const { headerGroups, rows, prepareRow } = useTable(
        {
            columns,
            data: simulatedNetworkBaselines,
            initialState: {
                sortBy: [
                    {
                        id: 'status',
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
