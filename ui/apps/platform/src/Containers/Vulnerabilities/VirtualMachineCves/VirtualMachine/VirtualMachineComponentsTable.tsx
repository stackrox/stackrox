import React from 'react';
import { Td, Thead, Tr, Tbody, Th, Table } from '@patternfly/react-table';

import type { CveComponentRow } from '../aggregateUtils';
import AdvisoryLinkOrText from '../../components/AdvisoryLinkOrText';
import FixedByVersion from '../../WorkloadCves/components/FixedByVersion';

export type VirtualMachineComponentsTableProps = {
    components: CveComponentRow[];
};

function VirtualMachineComponentsTable({ components }: VirtualMachineComponentsTableProps) {
    return (
        <Table style={{ border: '1px solid var(--pf-v5-c-table--BorderColor)' }}>
            <Thead noWrap>
                <Tr>
                    <Th>Component</Th>
                    <Th>Type</Th>
                    <Th>Version</Th>
                    <Th>CVE fixed in</Th>
                    <Th>Advisory</Th>
                </Tr>
            </Thead>
            <Tbody>
                {components.map(({ name, version, advisory, fixedBy, sourceType }) => {
                    return (
                        <Tr key={name}>
                            <Td dataLabel="Component">{name}</Td>
                            <Td dataLabel="Type">{sourceType}</Td>
                            <Td dataLabel="Version">{version}</Td>
                            <Td dataLabel="CVE fixed in">
                                <FixedByVersion fixedByVersion={fixedBy ?? ''} />
                            </Td>
                            <Td dataLabel="Advisory">
                                <AdvisoryLinkOrText advisory={advisory} />
                            </Td>
                        </Tr>
                    );
                })}
            </Tbody>
        </Table>
    );
}

export default VirtualMachineComponentsTable;
