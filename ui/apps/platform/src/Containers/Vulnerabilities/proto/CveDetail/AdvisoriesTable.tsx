import { Label } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import type { ProtoAdvisory } from './useCveDetail';

function severityColor(severity: string): 'red' | 'orange' | 'blue' | 'grey' {
    switch (severity.toUpperCase()) {
        case 'CRITICAL_VULNERABILITY_SEVERITY':
        case 'CRITICAL':
            return 'red';
        case 'IMPORTANT_VULNERABILITY_SEVERITY':
        case 'IMPORTANT':
        case 'HIGH':
            return 'orange';
        case 'MODERATE_VULNERABILITY_SEVERITY':
        case 'MODERATE':
        case 'MEDIUM':
            return 'blue';
        default:
            return 'grey';
    }
}

function severityLabel(severity: string): string {
    return severity
        .replace('_VULNERABILITY_SEVERITY', '')
        .replace(/_/g, ' ')
        .toLowerCase()
        .replace(/^\w/, (c) => c.toUpperCase());
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
                    <Th>Fixable</Th>
                    <Th>Fixed By</Th>
                    <Th>Published</Th>
                </Tr>
            </Thead>
            <Tbody>
                {advisories.map((adv) => (
                    <Tr key={adv.id}>
                        <Td dataLabel="Advisory ID">{adv.advisoryId}</Td>
                        <Td dataLabel="Severity">
                            <Label color={severityColor(adv.severity)}>
                                {severityLabel(adv.severity)}
                            </Label>
                        </Td>
                        <Td dataLabel="CVSS">{adv.cvss.toFixed(1)}</Td>
                        <Td dataLabel="Source">{adv.source}</Td>
                        <Td dataLabel="Fixable">{adv.fixable ? 'Yes' : 'No'}</Td>
                        <Td dataLabel="Fixed By">{adv.fixedBy || '-'}</Td>
                        <Td dataLabel="Published">
                            {adv.publishedDate
                                ? new Date(adv.publishedDate).toLocaleDateString()
                                : '-'}
                        </Td>
                    </Tr>
                ))}
                {advisories.length === 0 && (
                    <Tr>
                        <Td colSpan={7}>No advisories found</Td>
                    </Tr>
                )}
            </Tbody>
        </Table>
    );
}

export default AdvisoriesTable;
