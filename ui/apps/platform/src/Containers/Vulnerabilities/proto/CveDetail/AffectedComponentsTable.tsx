import { Bullseye } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Link } from 'react-router-dom-v5-compat';

import { vulnerabilitiesPrototypeComponentsPath } from 'routePaths';

import type { ProtoComponent } from './useCveDetail';
import { TABLE_HEADER_STYLE, TABLE_CELL_STYLE } from '../utils/tableDefaults';

type AffectedComponentsTableProps = {
    components: ProtoComponent[];
};

/**
 * Displays a table of affected components for a given CVE.
 */
function AffectedComponentsTable({ components }: AffectedComponentsTableProps) {
    return (
        <Table aria-label="Affected components" variant="compact">
            <Thead style={{ borderBottom: '2px solid var(--pf-global--BorderColor--100)' }}>
                <Tr>
                    <Th style={TABLE_HEADER_STYLE}>Component</Th>
                    <Th style={TABLE_HEADER_STYLE}>Version</Th>
                    <Th style={TABLE_HEADER_STYLE}>Source</Th>
                    <Th style={TABLE_HEADER_STYLE}>Fixed By</Th>
                    <Th style={TABLE_HEADER_STYLE}>Images</Th>
                </Tr>
            </Thead>
            <Tbody>
                {components.map((comp) => (
                    <Tr key={`${comp.name}-${comp.version}-${comp.source}`}>
                        <Td dataLabel="Component" style={TABLE_CELL_STYLE}>
                            <Link to={`${vulnerabilitiesPrototypeComponentsPath}/${encodeURIComponent(comp.name)}`}>
                                {comp.name}
                            </Link>
                        </Td>
                        <Td dataLabel="Version" style={TABLE_CELL_STYLE}>{comp.version}</Td>
                        <Td dataLabel="Source" style={TABLE_CELL_STYLE}>{comp.source}</Td>
                        <Td dataLabel="Fixed By" style={TABLE_CELL_STYLE}>{comp.fixedBy || '-'}</Td>
                        <Td dataLabel="Images" style={TABLE_CELL_STYLE}>{comp.imageCount}</Td>
                    </Tr>
                ))}
                {components.length === 0 && (
                    <Tr>
                        <Td colSpan={5} style={TABLE_CELL_STYLE}>
                            <Bullseye>No affected components found</Bullseye>
                        </Td>
                    </Tr>
                )}
            </Tbody>
        </Table>
    );
}

export default AffectedComponentsTable;
