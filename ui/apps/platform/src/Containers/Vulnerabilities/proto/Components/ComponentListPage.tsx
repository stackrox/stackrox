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

import { vulnerabilitiesPrototypeComponentsPath } from 'routePaths';

import ProtoNav from '../ProtoNav';
import { useComponentList } from './useComponentList';
import type { ProtoComponentListItem } from './useComponentList';

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

type SeverityBadgeProps = {
    label: string;
    count: number;
    color: 'red' | 'orange' | 'blue' | 'gold' | 'grey';
};

/**
 * Renders a single severity count badge like "C:5".
 */
function SeverityBadge({ label, count, color }: SeverityBadgeProps) {
    if (count === 0) {
        return null;
    }
    return (
        <Label color={color} isCompact style={{ marginRight: '4px' }}>
            {label}:{count}
        </Label>
    );
}

/**
 * Renders severity breakdown badges for a component row.
 */
function SeverityBreakdown({ component }: { component: ProtoComponentListItem }) {
    const badges: SeverityBadgeProps[] = [
        { label: 'C', count: component.criticalCount, color: 'red' },
        { label: 'I', count: component.importantCount, color: 'orange' },
        { label: 'M', count: component.moderateCount, color: 'blue' },
        { label: 'L', count: component.lowCount, color: 'gold' },
    ];

    const hasBadges = badges.some((b) => b.count > 0);
    if (!hasBadges) {
        return <>0</>;
    }

    return (
        <>
            {badges.map((b) => (
                <SeverityBadge key={b.label} {...b} />
            ))}
        </>
    );
}

function ComponentListPage() {
    const { data, loading, error } = useComponentList(50, 0);

    const components: ProtoComponentListItem[] = data?.components ?? [];
    const totalCount = data?.totalCount ?? 0;

    return (
        <>
            <PageSection hasBodyWrapper={false}>
                <Title headingLevel="h1">Vuln Management V5</Title>
            </PageSection>
            <PageSection hasBodyWrapper={false}>
                <ProtoNav />
            </PageSection>
            <PageSection hasBodyWrapper={false}>
                <Toolbar>
                    <ToolbarContent>
                        <ToolbarItem>
                            {loading && <Spinner size="md" />}
                            {!loading &&
                                `${components.length} of ${totalCount} Components`}
                        </ToolbarItem>
                    </ToolbarContent>
                </Toolbar>

                {error && (
                    <Bullseye>
                        <p>Error loading components: {error.message}</p>
                    </Bullseye>
                )}

                <Table aria-label="Vuln Management V5 component list" variant="compact">
                    <Thead>
                        <Tr>
                            <Th>Component</Th>
                            <Th>Versions</Th>
                            <Th info={{ tooltip: 'CVE counts by severity: Critical, Important, Moderate, Low' }}>CVEs</Th>
                            <Th>Images</Th>
                            <Th>Top Severity</Th>
                            <Th>Top CVSS</Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {components.map((comp) => (
                            <Tr key={comp.name}>
                                <Td dataLabel="Component">
                                    <Link
                                        to={`${vulnerabilitiesPrototypeComponentsPath}/${encodeURIComponent(comp.name)}`}
                                    >
                                        {comp.name}
                                    </Link>
                                </Td>
                                <Td dataLabel="Versions">{comp.versionCount}</Td>
                                <Td dataLabel="CVEs">
                                    <SeverityBreakdown component={comp} />
                                </Td>
                                <Td dataLabel="Images">{comp.imageCount}</Td>
                                <Td dataLabel="Top Severity">
                                    <Label color={severityColor(comp.topSeverity)}>
                                        {severityLabel(comp.topSeverity)}
                                    </Label>
                                </Td>
                                <Td dataLabel="Top CVSS">
                                    {formatCvss(comp.topCvss)}
                                </Td>
                            </Tr>
                        ))}
                        {!loading && components.length === 0 && (
                            <Tr>
                                <Td colSpan={6}>
                                    <Bullseye>No components found</Bullseye>
                                </Td>
                            </Tr>
                        )}
                    </Tbody>
                </Table>
            </PageSection>
        </>
    );
}

export default ComponentListPage;
