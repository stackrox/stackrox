import React from 'react';
import { Table, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { NetworkBaselinePeerStatus } from 'types/networkBaseline.proto';
import { TableUIState } from 'utils/getTableUIState';

import { getFlowKey } from '../utils/flowUtils';

type FlowTableProps = {
    emptyStateMessage: string;
    tableState: TableUIState<NetworkBaselinePeerStatus>;
};

export function FlowTable({ emptyStateMessage, tableState }: FlowTableProps) {
    return (
        <>
            <Table variant="compact">
                <Thead>
                    <Tr>
                        <Th>Entity</Th>
                        <Th>Direction</Th>
                        <Th>Port / protocol</Th>
                    </Tr>
                </Thead>

                <TbodyUnified
                    tableState={tableState}
                    colSpan={3}
                    emptyProps={{ message: emptyStateMessage }}
                    renderer={({ data }) => (
                        <Tbody>
                            {data.map((flow) => (
                                <Tr key={getFlowKey(flow)}>
                                    <Td>{flow.peer.entity.name}</Td>
                                    <Td>{flow.peer.ingress ? 'Ingress' : 'Egress'}</Td>
                                    <Td>{`${flow.peer.port} / ${
                                        flow.peer.protocol === 'L4_PROTOCOL_TCP' ? 'TCP' : 'UDP'
                                    }`}</Td>
                                </Tr>
                            ))}
                        </Tbody>
                    )}
                />
            </Table>
        </>
    );
}
