import { useState } from 'react';
import { useParams } from 'react-router-dom-v5-compat';
import {
    Bullseye,
    Label,
    PageSection,
    Spinner,
    Tab,
    Tabs,
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

import {
    vulnerabilitiesPrototypePath,
    vulnerabilitiesPrototypeComponentsPath,
    vulnerabilitiesPrototypeImageDetailPath,
} from 'routePaths';

import { useComponentDetail } from './useComponentDetail';
import { useComponentImages } from './useComponentImages';
import { useComponentCVEs } from './useComponentCVEs';
import type { ProtoComponentVersion } from './useComponentDetail';
import type { ProtoComponentImage } from './useComponentImages';
import type { ComponentCVE } from './useComponentCVEs';
import { DetailPageLayout } from '../components/DetailPageLayout';
import { TABLE_HEADER_STYLE, TABLE_CELL_STYLE } from '../utils/tableDefaults';

const severityNames: Record<number, string> = {
    0: 'Unknown',
    1: 'Low',
    2: 'Moderate',
    3: 'Important',
    4: 'Critical',
};

function severityColor(
    severity: number
): 'red' | 'orange' | 'blue' | 'yellow' | 'grey' {
    switch (severity) {
        case 4:
            return 'red';
        case 3:
            return 'orange';
        case 2:
            return 'blue';
        case 1:
            return 'yellow';
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

function imageDetailPath(imageId: string): string {
    return vulnerabilitiesPrototypeImageDetailPath.replace(':imageId', encodeURIComponent(imageId));
}

/**
 * Sub-table rendered inside an expanded version row, showing CVEs
 * for that component version.
 */
function CveSubTable({
    componentName,
    componentVersion,
}: {
    componentName: string;
    componentVersion: string;
}) {
    const { data: cves, loading, error } = useComponentCVEs(componentName, componentVersion, true);

    if (loading) {
        return (
            <Bullseye>
                <Spinner size="md" />
            </Bullseye>
        );
    }

    if (error) {
        return <p>Error loading CVEs: {error.message}</p>;
    }

    if (cves.length === 0) {
        return <Bullseye>No CVEs for this version</Bullseye>;
    }

    return (
        <Table aria-label="Version CVEs" variant="compact" borders={false}>
            <Thead>
                <Tr>
                    <Th style={TABLE_HEADER_STYLE}>CVE</Th>
                    <Th style={TABLE_HEADER_STYLE}>Severity</Th>
                    <Th style={TABLE_HEADER_STYLE}>CVSS</Th>
                    <Th style={TABLE_HEADER_STYLE}>Fixable</Th>
                    <Th style={TABLE_HEADER_STYLE}>Fixed By</Th>
                    <Th style={TABLE_HEADER_STYLE}>Images</Th>
                </Tr>
            </Thead>
            <Tbody>
                {cves.map((cve: ComponentCVE) => (
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
                                {severityLabel(cve.severity)}
                            </Label>
                        </Td>
                        <Td dataLabel="CVSS" style={TABLE_CELL_STYLE}>{formatCvss(cve.cvss)}</Td>
                        <Td dataLabel="Fixable" style={TABLE_CELL_STYLE}>{cve.fixable ? 'Yes' : 'No'}</Td>
                        <Td dataLabel="Fixed By" style={TABLE_CELL_STYLE}>{cve.fixedBy || '-'}</Td>
                        <Td dataLabel="Images" style={TABLE_CELL_STYLE}>{cve.imageCount}</Td>
                    </Tr>
                ))}
            </Tbody>
        </Table>
    );
}

function VersionsTable({
    versions,
    loading,
    componentName,
}: {
    versions: ProtoComponentVersion[];
    loading: boolean;
    componentName: string;
}) {
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

    const columnCount = 9;

    return (
        <Table aria-label="Component versions" variant="compact">
            <Thead style={{ borderBottom: '2px solid var(--pf-global--BorderColor--100)' }}>
                <Tr>
                    <Th screenReaderText="Row expansion" style={TABLE_HEADER_STYLE} />
                    <Th style={TABLE_HEADER_STYLE}>Version</Th>
                    <Th style={TABLE_HEADER_STYLE}>Source</Th>
                    <Th style={TABLE_HEADER_STYLE}>CVEs</Th>
                    <Th style={TABLE_HEADER_STYLE}>Images</Th>
                    <Th style={TABLE_HEADER_STYLE}>Top Severity</Th>
                    <Th style={TABLE_HEADER_STYLE}>Top CVSS</Th>
                    <Th style={TABLE_HEADER_STYLE}>Fixable</Th>
                    <Th style={TABLE_HEADER_STYLE}>Fixed By</Th>
                </Tr>
            </Thead>
            {versions.map((ver, rowIndex) => {
                const rowKey = ver.version;
                const isExpanded = expandedRows.has(rowKey);
                return (
                    <Tbody key={rowKey} isExpanded={isExpanded}>
                        <Tr>
                            <Td
                                expand={
                                    ver.cveCount > 0
                                        ? {
                                              rowIndex,
                                              isExpanded,
                                              onToggle: () => toggleExpand(rowKey),
                                          }
                                        : undefined
                                }
                                style={TABLE_CELL_STYLE}
                            />
                            <Td dataLabel="Version" style={TABLE_CELL_STYLE}>{ver.version}</Td>
                            <Td dataLabel="Source" style={TABLE_CELL_STYLE}>{ver.source}</Td>
                            <Td dataLabel="CVEs" style={TABLE_CELL_STYLE}>{ver.cveCount}</Td>
                            <Td dataLabel="Images" style={TABLE_CELL_STYLE}>{ver.imageCount}</Td>
                            <Td dataLabel="Top Severity" style={TABLE_CELL_STYLE}>
                                <Label color={severityColor(ver.topSeverity)}>
                                    {severityLabel(ver.topSeverity)}
                                </Label>
                            </Td>
                            <Td dataLabel="Top CVSS" style={TABLE_CELL_STYLE}>{formatCvss(ver.topCvss)}</Td>
                            <Td dataLabel="Fixable" style={TABLE_CELL_STYLE}>{ver.fixable ? 'Yes' : 'No'}</Td>
                            <Td dataLabel="Fixed By" style={TABLE_CELL_STYLE}>{ver.fixedBy || '-'}</Td>
                        </Tr>
                        {ver.cveCount > 0 && (
                            <Tr isExpanded={isExpanded}>
                                <Td colSpan={columnCount} style={TABLE_CELL_STYLE}>
                                    <ExpandableRowContent>
                                        {isExpanded && (
                                            <CveSubTable
                                                componentName={componentName}
                                                componentVersion={ver.version}
                                            />
                                        )}
                                    </ExpandableRowContent>
                                </Td>
                            </Tr>
                        )}
                    </Tbody>
                );
            })}
            {!loading && versions.length === 0 && (
                <Tbody>
                    <Tr>
                        <Td colSpan={columnCount} style={TABLE_CELL_STYLE}>
                            <Bullseye>No versions found for this component</Bullseye>
                        </Td>
                    </Tr>
                </Tbody>
            )}
        </Table>
    );
}

function ImagesTable({
    images,
    loading,
}: {
    images: ProtoComponentImage[];
    loading: boolean;
}) {
    return (
        <Table aria-label="Component images" variant="compact">
            <Thead style={{ borderBottom: '2px solid var(--pf-global--BorderColor--100)' }}>
                <Tr>
                    <Th style={TABLE_HEADER_STYLE}>Image</Th>
                    <Th style={TABLE_HEADER_STYLE}>Version</Th>
                    <Th style={TABLE_HEADER_STYLE}>Arch</Th>
                    <Th style={TABLE_HEADER_STYLE}>CVEs</Th>
                    <Th style={TABLE_HEADER_STYLE}>Top Severity</Th>
                    <Th style={TABLE_HEADER_STYLE}>Fixable</Th>
                </Tr>
            </Thead>
            <Tbody>
                {images.map((img) => {
                    const displayName = img.imageName || img.imageId;
                    return (
                        <Tr key={`${img.imageId}-${img.version}`}>
                            <Td dataLabel="Image" style={TABLE_CELL_STYLE}>
                                <Link to={imageDetailPath(img.imageId)}>{displayName}</Link>
                            </Td>
                            <Td dataLabel="Version" style={TABLE_CELL_STYLE}>{img.version}</Td>
                            <Td dataLabel="Arch" style={TABLE_CELL_STYLE}>{img.arch || '-'}</Td>
                            <Td dataLabel="CVEs" style={TABLE_CELL_STYLE}>{img.cveCount}</Td>
                            <Td dataLabel="Top Severity" style={TABLE_CELL_STYLE}>
                                <Label color={severityColor(img.topSeverity)}>
                                    {severityLabel(img.topSeverity)}
                                </Label>
                            </Td>
                            <Td dataLabel="Fixable" style={TABLE_CELL_STYLE}>{img.fixable ? 'Yes' : 'No'}</Td>
                        </Tr>
                    );
                })}
                {!loading && images.length === 0 && (
                    <Tr>
                        <Td colSpan={6} style={TABLE_CELL_STYLE}>
                            <Bullseye>No images found for this component</Bullseye>
                        </Td>
                    </Tr>
                )}
            </Tbody>
        </Table>
    );
}

function ComponentDetailPage() {
    const { componentName } = useParams<{ componentName: string }>();
    const decodedName = componentName ? decodeURIComponent(componentName) : '';
    const { data, loading, error } = useComponentDetail(decodedName);
    const {
        data: images,
        loading: imagesLoading,
        error: imagesError,
    } = useComponentImages(decodedName);

    const [activeTab, setActiveTab] = useState<string | number>('versions');

    const versions: ProtoComponentVersion[] = data?.versions ?? [];
    const totalCveCount = versions.reduce((sum, v) => sum + v.cveCount, 0);
    const totalImageCount = versions.reduce((sum, v) => sum + v.imageCount, 0);

    const breadcrumbs = [
        { label: 'Vulnerability Management V5', path: vulnerabilitiesPrototypePath },
        { label: 'Components', path: vulnerabilitiesPrototypeComponentsPath },
        { label: data?.name ?? decodedName },
    ];

    const subtitle = `${versions.length} versions • ${totalCveCount} CVEs`;

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
                <p>Error loading component: {error.message}</p>
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
                title={data.name}
                subtitle={subtitle}
            >
                <Tabs
                    activeKey={activeTab}
                    onSelect={(_event, tabKey) => setActiveTab(tabKey)}
                    aria-label="Component detail sections"
                >
                    <Tab eventKey="versions" title={`Versions (${versions.length})`}>
                        <VersionsTable versions={versions} loading={loading} componentName={decodedName} />
                    </Tab>
                    <Tab eventKey="images" title={`Images (${images.length})`}>
                        {imagesLoading && (
                            <Bullseye>
                                <Spinner />
                            </Bullseye>
                        )}
                        {imagesError && <p>Error loading images: {imagesError.message}</p>}
                        {!imagesLoading && !imagesError && (
                            <ImagesTable images={images} loading={imagesLoading} />
                        )}
                    </Tab>
                </Tabs>
            </DetailPageLayout>
        </PageSection>
    );
}

export default ComponentDetailPage;
