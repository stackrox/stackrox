import { Bullseye, Label, Truncate } from '@patternfly/react-core';
import { ExternalLinkAltIcon } from '@patternfly/react-icons';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import type { ProtoAdvisory } from './useCveDetail';

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
            <Thead>
                <Tr>
                    <Th>Advisory ID</Th>
                    <Th>Severity</Th>
                    <Th>CVSS</Th>
                    <Th>Source</Th>
                    <Th>Fixed By</Th>
                    <Th>Description</Th>
                    <Th>Link</Th>
                </Tr>
            </Thead>
            <Tbody>
                {advisories.map((adv) => (
                    <Tr key={adv.id}>
                        <Td dataLabel="Advisory ID">{adv.id}</Td>
                        <Td dataLabel="Severity">
                            <Label color={severityColor(adv.severity)}>
                                {severityLabel(adv.severity)}
                            </Label>
                        </Td>
                        <Td dataLabel="CVSS">
                            {adv.cvss ? adv.cvss.toFixed(1) : '-'}
                        </Td>
                        <Td dataLabel="Source">{adv.sourceName}</Td>
                        <Td dataLabel="Fixed By">{adv.fixedBy || '-'}</Td>
                        <Td dataLabel="Description" modifier="truncate">
                            {adv.description ? (
                                <Truncate
                                    content={adv.description}
                                    tooltipPosition="top"
                                />
                            ) : (
                                '-'
                            )}
                        </Td>
                        <Td dataLabel="Link">
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
                        <Td colSpan={7}>
                            <Bullseye>No advisories found</Bullseye>
                        </Td>
                    </Tr>
                )}
            </Tbody>
        </Table>
    );
}

export default AdvisoriesTable;
