import React from 'react';
import { Td, Thead, Tr, Tbody, Th, Table } from '@patternfly/react-table';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';

import type { CveComponentRow } from '../aggregateUtils';

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
                            <Td dataLabel="CVE fixed in">{fixedBy}</Td>
                            <Td dataLabel="Advisory">
                                <ExternalLink>
                                    <a
                                        href={advisory.link}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                    >
                                        {advisory.name}
                                    </a>
                                </ExternalLink>
                            </Td>
                        </Tr>
                    );
                })}
            </Tbody>
        </Table>
    );
}

export default VirtualMachineComponentsTable;
