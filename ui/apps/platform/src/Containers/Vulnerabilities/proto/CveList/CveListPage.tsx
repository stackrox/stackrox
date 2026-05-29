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

function formatCvss(cvss: number): string {
    return cvss ? cvss.toFixed(1) : '-';
}

function formatDate(dateStr: string | null): string {
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

    const cves: ProtoCVEListItem[] = data?.cves ?? [];
    const totalCount = data?.totalCount ?? 0;

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
                            {!loading &&
                                `${cves.length} of ${totalCount} CVEs`}
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
                            <Th info={{ tooltip: 'Highest severity reported by any advisory for this CVE' }}>Top Severity</Th>
                            <Th info={{ tooltip: 'Highest CVSS score reported by any advisory for this CVE' }}>Top CVSS</Th>
                            <Th info={{ tooltip: 'Number of distinct images affected by this CVE' }}>Images</Th>
                            <Th info={{ tooltip: 'Whether a fix is available from any advisory source' }}>Fixable</Th>
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
                                <Td dataLabel="CVSS">
                                    {formatCvss(cve.cvss)}
                                </Td>
                                <Td dataLabel="Images">{cve.imageCount}</Td>
                                <Td dataLabel="Fixable">
                                    {cve.fixable ? 'Yes' : 'No'}
                                </Td>
                                <Td dataLabel="First Seen">
                                    {formatDate(cve.firstSeen)}
                                </Td>
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
