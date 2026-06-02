import { Link } from 'react-router-dom-v5-compat';
import {
    Bullseye,
    Label,
    Pagination,
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
import { usePagination } from '../usePagination';
import { useSort } from '../useSort';
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
    color: 'red' | 'orange' | 'blue' | 'yellow' | 'grey';
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
        { label: 'L', count: component.lowCount, color: 'yellow' },
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

// Column keys: name, versions (not sortable), cveCount, imageCount, severity, topCvss (not sortable)
const compSortColumns = ['name', '', 'cveCount', 'imageCount', 'severity', ''];

function ComponentListPage() {
    const { sortBy, sortDir, getThSortProps } = useSort(compSortColumns, 4);
    const { page, perPage, offset, onSetPage, onPerPageSelect } = usePagination(20);
    const { data, loading, error } = useComponentList(perPage, offset, sortBy, sortDir);

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
                            <Th {...getThSortProps(0)}>Component</Th>
                            <Th>Versions</Th>
                            <Th {...getThSortProps(2)} info={{ tooltip: 'CVE counts by severity: Critical, Important, Moderate, Low' }}>CVEs</Th>
                            <Th {...getThSortProps(3)}>Images</Th>
                            <Th {...getThSortProps(4)}>Top Severity</Th>
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
                <Pagination
                    itemCount={totalCount}
                    perPage={perPage}
                    page={page}
                    onSetPage={onSetPage}
                    onPerPageSelect={onPerPageSelect}
                />
            </PageSection>
        </>
    );
}

export default ComponentListPage;
