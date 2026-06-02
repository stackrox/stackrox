import { Bullseye, Label, Truncate } from '@patternfly/react-core';
import { ExternalLinkAltIcon } from '@patternfly/react-icons';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import type { ProtoAdvisory } from './useCveDetail';
import { TABLE_HEADER_STYLE, TABLE_CELL_STYLE } from '../utils/tableDefaults';

const severityNames: Record<number, string> = {
    0: 'Unknown',
    1: 'Low',
    2: 'Moderate',
    3: 'Important',
    4: 'Critical',
};

function severityColor(severity: number): 'red' | 'orange' | 'blue' | 'grey' {
    switch (severity) {
        case 4:
            return 'red';
        case 3:
            return 'orange';
        case 2:
            return 'blue';
        default:
            return 'grey';
    }
}

function severityLabel(severity: number): string {
    return severityNames[severity] ?? 'Unknown';
}

type AdvisoriesTableProps = {
    advisories: ProtoAdvisory[];
};

/**
 * Displays a table of advisories for a given CVE.
 */
function AdvisoriesTable({ advisories }: AdvisoriesTableProps) {
    return (
        <Table aria-label="Advisories" variant="compact">
            <Thead style={{ borderBottom: '2px solid var(--pf-global--BorderColor--100)' }}>
                <Tr>
                    <Th style={TABLE_HEADER_STYLE}>Advisory ID</Th>
                    <Th style={TABLE_HEADER_STYLE}>Severity</Th>
                    <Th style={TABLE_HEADER_STYLE}>CVSS</Th>
                    <Th style={TABLE_HEADER_STYLE}>Source</Th>
                    <Th style={TABLE_HEADER_STYLE}>Fixed By</Th>
                    <Th style={TABLE_HEADER_STYLE}>Description</Th>
                    <Th style={TABLE_HEADER_STYLE}>Link</Th>
                </Tr>
            </Thead>
            <Tbody>
                {advisories.map((adv) => (
                    <Tr key={adv.id}>
                        <Td dataLabel="Advisory ID" style={TABLE_CELL_STYLE}>{adv.id}</Td>
                        <Td dataLabel="Severity" style={TABLE_CELL_STYLE}>
                            <Label color={severityColor(adv.severity)}>
                                {severityLabel(adv.severity)}
                            </Label>
                        </Td>
                        <Td dataLabel="CVSS" style={TABLE_CELL_STYLE}>
                            {adv.cvss ? adv.cvss.toFixed(1) : '-'}
                        </Td>
                        <Td dataLabel="Source" style={TABLE_CELL_STYLE}>{adv.sourceName}</Td>
                        <Td dataLabel="Fixed By" style={TABLE_CELL_STYLE}>{adv.fixedBy || '-'}</Td>
                        <Td dataLabel="Description" modifier="truncate" style={TABLE_CELL_STYLE}>
                            {adv.description ? (
                                <Truncate
                                    content={adv.description}
                                    tooltipPosition="top"
                                />
                            ) : (
                                '-'
                            )}
                        </Td>
                        <Td dataLabel="Link" style={TABLE_CELL_STYLE}>
                            {adv.link ? (
                                <a
                                    href={adv.link}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                >
                                    View <ExternalLinkAltIcon />
                                </a>
                            ) : (
                                '-'
                            )}
                        </Td>
                    </Tr>
                ))}
                {advisories.length === 0 && (
                    <Tr>
                        <Td colSpan={7} style={TABLE_CELL_STYLE}>
                            <Bullseye>No advisories found</Bullseye>
                        </Td>
                    </Tr>
                )}
            </Tbody>
        </Table>
    );
}

export default AdvisoriesTable;
