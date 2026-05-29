import { Bullseye, Label } from '@patternfly/react-core';
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
                    </Tr>
                ))}
                {advisories.length === 0 && (
                    <Tr>
                        <Td colSpan={4}>
                            <Bullseye>No advisories found</Bullseye>
                        </Td>
                    </Tr>
                )}
            </Tbody>
        </Table>
    );
}

export default AdvisoriesTable;
