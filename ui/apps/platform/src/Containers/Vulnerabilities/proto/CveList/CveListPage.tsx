import { Link } from 'react-router-dom-v5-compat';
import {
    Bullseye,
    Label,
    PageSection,
    Spinner,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import { vulnerabilitiesPrototypeCvePath } from 'routePaths';

import { useCveList } from './useCveList';
import type { ProtoCVEListItem } from './useCveList';

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

function formatDate(dateStr: string): string {
    if (!dateStr) {
        return '-';
    }
    try {
        return new Date(dateStr).toLocaleDateString();
    } catch {
        return dateStr;
    }
}

function CveListPage() {
    const { data, loading, error } = useCveList(100, 0);

    const cves: ProtoCVEListItem[] = data?.protoCVEList ?? [];

    return (
        <>
            <PageSection hasBodyWrapper={false}>
                <Title headingLevel="h1">CVE Prototype — List</Title>
            </PageSection>
            <PageSection hasBodyWrapper={false}>
                <Toolbar>
                    <ToolbarContent>
                        <ToolbarItem>
                            {loading && <Spinner size="md" />}
                            {!loading && `${cves.length} CVEs`}
                        </ToolbarItem>
                    </ToolbarContent>
                </Toolbar>

                {error && (
                    <Bullseye>
                        <p>Error loading CVEs: {error.message}</p>
                    </Bullseye>
                )}

                <Table aria-label="Prototype CVE list" variant="compact">
                    <Thead>
                        <Tr>
                            <Th>CVE</Th>
                            <Th>Severity</Th>
                            <Th>CVSS</Th>
                            <Th>Images</Th>
                            <Th>Fixable</Th>
                            <Th>First Seen</Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {cves.map((cve) => (
                            <Tr key={cve.cveName}>
                                <Td dataLabel="CVE">
                                    <Link
                                        to={`${vulnerabilitiesPrototypeCvePath}/${encodeURIComponent(cve.cveName)}`}
                                    >
                                        {cve.cveName}
                                    </Link>
                                </Td>
                                <Td dataLabel="Severity">
                                    <Label color={severityColor(cve.severity)}>
                                        {severityLabel(cve.severity)}
                                    </Label>
                                </Td>
                                <Td dataLabel="CVSS">{cve.cvss.toFixed(1)}</Td>
                                <Td dataLabel="Images">{cve.imageCount}</Td>
                                <Td dataLabel="Fixable">{cve.fixable ? 'Yes' : 'No'}</Td>
                                <Td dataLabel="First Seen">{formatDate(cve.firstSeen)}</Td>
                            </Tr>
                        ))}
                        {!loading && cves.length === 0 && (
                            <Tr>
                                <Td colSpan={6}>
                                    <Bullseye>No CVEs found</Bullseye>
                                </Td>
                            </Tr>
                        )}
                    </Tbody>
                </Table>
            </PageSection>
        </>
    );
}

export default CveListPage;
