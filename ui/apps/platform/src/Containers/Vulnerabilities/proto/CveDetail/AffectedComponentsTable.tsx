import { Bullseye } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import type { ProtoComponent } from './useCveDetail';

type AffectedComponentsTableProps = {
    components: ProtoComponent[];
};

/**
 * Displays a table of affected components for a given CVE.
 */
function AffectedComponentsTable({ components }: AffectedComponentsTableProps) {
    return (
        <Table aria-label="Affected components" variant="compact">
            <Thead>
                <Tr>
                    <Th>Component</Th>
                    <Th>Version</Th>
                    <Th>Source</Th>
                    <Th>Fixed By</Th>
                    <Th>Images</Th>
                </Tr>
            </Thead>
            <Tbody>
                {components.map((comp) => (
                    <Tr key={`${comp.name}-${comp.version}-${comp.source}`}>
                        <Td dataLabel="Component">{comp.name}</Td>
                        <Td dataLabel="Version">{comp.version}</Td>
                        <Td dataLabel="Source">{comp.source}</Td>
                        <Td dataLabel="Fixed By">{comp.fixedBy || '-'}</Td>
                        <Td dataLabel="Images">{comp.imageCount}</Td>
                    </Tr>
                ))}
                {components.length === 0 && (
                    <Tr>
                        <Td colSpan={5}>
                            <Bullseye>No affected components found</Bullseye>
                        </Td>
                    </Tr>
                )}
            </Tbody>
        </Table>
    );
}

export default AffectedComponentsTable;
