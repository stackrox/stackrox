import { useState } from 'react';
import { useParams } from 'react-router-dom-v5-compat';
import {
    Bullseye,
    Flex,
    FlexItem,
    Label,
    PageSection,
    Spinner,
    Stack,
    StackItem,
    Title,
} from '@patternfly/react-core';
import {
    ExpandableRowContent,
    Table,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';
import { Link } from 'react-router-dom-v5-compat';

import { vulnerabilitiesPrototypePath } from 'routePaths';

import ScanInfo from './ScanInfo';
import { useImageDetail } from './useImageDetail';
import type { ImageComponent, ImageCVE } from './useImageDetail';
import { DetailPageLayout } from '../components/DetailPageLayout';
import { TABLE_HEADER_STYLE, TABLE_CELL_STYLE, formatDate } from '../utils/tableDefaults';

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

function displayImageName(imageId: string, imageName?: string): string {
    if (imageName) {
        return imageName;
    }
    if (imageId.startsWith('sha256:') && imageId.length > 19) {
        return `${imageId.slice(0, 19)}...`;
    }
    return imageId;
}

type SummaryBadgeProps = {
    label: string;
    count: number;
    color: 'red' | 'orange' | 'blue' | 'grey' | 'yellow';
};

function SummaryBadge({ label, count, color }: SummaryBadgeProps) {
    return (
        <FlexItem>
            <Label color={color}>
                {label}: {count}
            </Label>
        </FlexItem>
    );
}

/**
 * Sub-table rendered inside an expanded component row, showing all CVEs
 * for that component.
 */
function CveSubTable({ cves }: { cves: ImageCVE[] }) {
    return (
        <Table
            aria-label="Component CVEs"
            variant="compact"
            borders={false}
        >
            <Thead>
                <Tr>
                    <Th style={TABLE_HEADER_STYLE}>CVE</Th>
                    <Th style={TABLE_HEADER_STYLE}>Severity</Th>
                    <Th style={TABLE_HEADER_STYLE}>CVSS</Th>
                    <Th style={TABLE_HEADER_STYLE}>Fixed By</Th>
                    <Th style={TABLE_HEADER_STYLE}>Advisories</Th>
                </Tr>
            </Thead>
            <Tbody>
                {cves.map((cve) => (
                    <Tr key={cve.cveName}>
                        <Td dataLabel="CVE" style={TABLE_CELL_STYLE}>
                            <Link
                                to={`/main/vulnerabilities/prototype/cves/${encodeURIComponent(cve.cveName)}`}
                            >
                                {cve.cveName}
                            </Link>
                        </Td>
                        <Td dataLabel="Severity" style={TABLE_CELL_STYLE}>
                            <Label color={severityColor(cve.severity)}>
                                {severityNames[cve.severity] ?? 'Unknown'}
                            </Label>
                        </Td>
                        <Td dataLabel="CVSS" style={TABLE_CELL_STYLE}>{cve.cvss.toFixed(1)}</Td>
                        <Td dataLabel="Fixed By" style={TABLE_CELL_STYLE}>{cve.fixedBy || '-'}</Td>
                        <Td dataLabel="Advisories" style={TABLE_CELL_STYLE}>
                            {cve.advisories?.join(', ') || '-'}
                        </Td>
                    </Tr>
                ))}
            </Tbody>
        </Table>
    );
}

/**
 * Components table with expandable rows. Each row shows a component;
 * expanding it reveals the CVEs affecting that component.
 */
function ComponentsTable({ components }: { components: ImageComponent[] }) {
    const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());

    function toggleExpand(key: string) {
        setExpandedRows((prev) => {
            const next = new Set(prev);
            if (next.has(key)) {
                next.delete(key);
            } else {
                next.add(key);
            }
            return next;
        });
    }

    const columnCount = 6;

    return (
        <Table aria-label="Image components" variant="compact">
            <Thead style={{ borderBottom: '2px solid var(--pf-global--BorderColor--100)' }}>
                <Tr>
                    <Th screenReaderText="Row expansion" style={TABLE_HEADER_STYLE} />
                    <Th style={TABLE_HEADER_STYLE}>Component</Th>
                    <Th style={TABLE_HEADER_STYLE}>Version</Th>
                    <Th style={TABLE_HEADER_STYLE}>Source</Th>
                    <Th style={TABLE_HEADER_STYLE}>Location</Th>
                    <Th style={TABLE_HEADER_STYLE}>CVEs</Th>
                </Tr>
            </Thead>
            {components.map((comp, rowIndex) => {
                const rowKey = `${comp.name}-${comp.version}`;
                const isExpanded = expandedRows.has(rowKey);
                return (
                    <Tbody key={rowKey} isExpanded={isExpanded}>
                        <Tr>
                            <Td
                                expand={{
                                    rowIndex,
                                    isExpanded,
                                    onToggle: () => toggleExpand(rowKey),
                                }}
                                style={TABLE_CELL_STYLE}
                            />
                            <Td dataLabel="Component" style={TABLE_CELL_STYLE}>{comp.name}</Td>
                            <Td dataLabel="Version" style={TABLE_CELL_STYLE}>{comp.version}</Td>
                            <Td dataLabel="Source" style={TABLE_CELL_STYLE}>{comp.source}</Td>
                            <Td dataLabel="Location" style={TABLE_CELL_STYLE}>
                                {comp.location || '-'}
                            </Td>
                            <Td dataLabel="CVEs" style={TABLE_CELL_STYLE}>{comp.cves.length}</Td>
                        </Tr>
                        <Tr isExpanded={isExpanded}>
                            <Td colSpan={columnCount} style={TABLE_CELL_STYLE}>
                                <ExpandableRowContent>
                                    {comp.cves.length > 0 ? (
                                        <CveSubTable cves={comp.cves} />
                                    ) : (
                                        <Bullseye>
                                            No CVEs for this component
                                        </Bullseye>
                                    )}
                                </ExpandableRowContent>
                            </Td>
                        </Tr>
                    </Tbody>
                );
            })}
            {components.length === 0 && (
                <Tbody>
                    <Tr>
                        <Td colSpan={columnCount} style={TABLE_CELL_STYLE}>
                            <Bullseye>No components found</Bullseye>
                        </Td>
                    </Tr>
                </Tbody>
            )}
        </Table>
    );
}

/**
 * Image detail page for the CVE prototype. Shows scan metadata,
 * a CVE severity summary, and an expandable components table.
 */
function ImageDetailPage() {
    const { imageId } = useParams<{ imageId: string }>();
    const { data, loading, error } = useImageDetail(imageId ?? '');

    const headerName = displayImageName(
        data?.imageId ?? imageId ?? '',
        data?.imageName
    );

    const breadcrumbs = [
        { label: 'Vulnerability Management V5', path: vulnerabilitiesPrototypePath },
        { label: 'Images', path: vulnerabilitiesPrototypePath + '/images' },
        { label: headerName },
    ];

    const componentCount = data?.components?.length ?? 0;
    const cveCount = data?.cveSummary?.total ?? 0;
    const subtitle = data?.scanTime
        ? `Scanned: ${formatDate(data.scanTime)} • ${componentCount} components • ${cveCount} CVEs`
        : `${componentCount} components • ${cveCount} CVEs`;

    if (loading) {
        return (
            <PageSection hasBodyWrapper={false}>
                <Bullseye>
                    <Spinner />
                </Bullseye>
            </PageSection>
        );
    }

    if (error) {
        return (
            <PageSection hasBodyWrapper={false}>
                <p>Error loading image detail: {error.message}</p>
            </PageSection>
        );
    }

    if (!data) {
        return null;
    }

    return (
        <PageSection hasBodyWrapper={false}>
            <DetailPageLayout
                breadcrumbs={breadcrumbs}
                title={headerName}
                subtitle={subtitle}
            >
                <Stack hasGutter>
                    <StackItem>
                        <ScanInfo
                            scannerVersion={data.scannerVersion}
                            bundleVersion={data.bundleVersion}
                            dataSources={data.dataSources}
                            scanTime={data.scanTime}
                        />
                    </StackItem>

                    <StackItem>
                        <Title headingLevel="h2">CVE Summary</Title>
                        <Flex
                            spaceItems={{
                                default: 'spaceItemsMd',
                            }}
                        >
                            <SummaryBadge
                                label="Critical"
                                count={data.cveSummary.critical}
                                color="red"
                            />
                            <SummaryBadge
                                label="Important"
                                count={data.cveSummary.important}
                                color="orange"
                            />
                            <SummaryBadge
                                label="Moderate"
                                count={data.cveSummary.moderate}
                                color="blue"
                            />
                            <SummaryBadge
                                label="Low"
                                count={data.cveSummary.low}
                                color="yellow"
                            />
                            <FlexItem>
                                <Label color="grey">
                                    Total: {data.cveSummary.total}
                                </Label>
                            </FlexItem>
                        </Flex>
                    </StackItem>

                    <StackItem>
                        <Title headingLevel="h2">Components</Title>
                        <ComponentsTable
                            components={data.components}
                        />
                    </StackItem>
                </Stack>
            </DetailPageLayout>
        </PageSection>
    );
}

export default ImageDetailPage;
