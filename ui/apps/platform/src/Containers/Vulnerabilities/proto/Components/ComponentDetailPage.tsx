import { useState } from 'react';
import { useParams } from 'react-router-dom-v5-compat';
import {
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Flex,
    FlexItem,
    Label,
    PageSection,
    Spinner,
    Tab,
    Tabs,
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

import {
    vulnerabilitiesPrototypeComponentsPath,
    vulnerabilitiesPrototypeImageDetailPath,
} from 'routePaths';

import { useComponentDetail } from './useComponentDetail';
import { useComponentImages } from './useComponentImages';
import { useComponentCVEs } from './useComponentCVEs';
import type { ProtoComponentVersion } from './useComponentDetail';
import type { ProtoComponentImage } from './useComponentImages';
import type { ComponentCVE } from './useComponentCVEs';

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
                    <Th>CVE</Th>
                    <Th>Severity</Th>
                    <Th>CVSS</Th>
                    <Th>Fixable</Th>
                    <Th>Fixed By</Th>
                    <Th>Images</Th>
                </Tr>
            </Thead>
            <Tbody>
                {cves.map((cve: ComponentCVE) => (
                    <Tr key={cve.cveName}>
                        <Td dataLabel="CVE">
                            <Link
                                to={`/main/vulnerabilities/prototype/cves/${encodeURIComponent(cve.cveName)}`}
                            >
                                {cve.cveName}
                            </Link>
                        </Td>
                        <Td dataLabel="Severity">
                            <Label color={severityColor(cve.severity)}>
                                {severityLabel(cve.severity)}
                            </Label>
                        </Td>
                        <Td dataLabel="CVSS">{formatCvss(cve.cvss)}</Td>
                        <Td dataLabel="Fixable">{cve.fixable ? 'Yes' : 'No'}</Td>
                        <Td dataLabel="Fixed By">{cve.fixedBy || '-'}</Td>
                        <Td dataLabel="Images">{cve.imageCount}</Td>
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
            <Thead>
                <Tr>
                    <Th screenReaderText="Row expansion" />
                    <Th>Version</Th>
                    <Th>Source</Th>
                    <Th>CVEs</Th>
                    <Th>Images</Th>
                    <Th>Top Severity</Th>
                    <Th>Top CVSS</Th>
                    <Th>Fixable</Th>
                    <Th>Fixed By</Th>
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
                            />
                            <Td dataLabel="Version">{ver.version}</Td>
                            <Td dataLabel="Source">{ver.source}</Td>
                            <Td dataLabel="CVEs">{ver.cveCount}</Td>
                            <Td dataLabel="Images">{ver.imageCount}</Td>
                            <Td dataLabel="Top Severity">
                                <Label color={severityColor(ver.topSeverity)}>
                                    {severityLabel(ver.topSeverity)}
                                </Label>
                            </Td>
                            <Td dataLabel="Top CVSS">{formatCvss(ver.topCvss)}</Td>
                            <Td dataLabel="Fixable">{ver.fixable ? 'Yes' : 'No'}</Td>
                            <Td dataLabel="Fixed By">{ver.fixedBy || '-'}</Td>
                        </Tr>
                        {ver.cveCount > 0 && (
                            <Tr isExpanded={isExpanded}>
                                <Td colSpan={columnCount}>
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
                        <Td colSpan={columnCount}>
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
            <Thead>
                <Tr>
                    <Th>Image</Th>
                    <Th>Version</Th>
                    <Th>Arch</Th>
                    <Th>CVEs</Th>
                    <Th>Top Severity</Th>
                    <Th>Fixable</Th>
                </Tr>
            </Thead>
            <Tbody>
                {images.map((img) => {
                    const displayName = img.imageName || img.imageId;
                    return (
                        <Tr key={`${img.imageId}-${img.version}`}>
                            <Td dataLabel="Image">
                                <Link to={imageDetailPath(img.imageId)}>{displayName}</Link>
                            </Td>
                            <Td dataLabel="Version">{img.version}</Td>
                            <Td dataLabel="Arch">{img.arch || '-'}</Td>
                            <Td dataLabel="CVEs">{img.cveCount}</Td>
                            <Td dataLabel="Top Severity">
                                <Label color={severityColor(img.topSeverity)}>
                                    {severityLabel(img.topSeverity)}
                                </Label>
                            </Td>
                            <Td dataLabel="Fixable">{img.fixable ? 'Yes' : 'No'}</Td>
                        </Tr>
                    );
                })}
                {!loading && images.length === 0 && (
                    <Tr>
                        <Td colSpan={6}>
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

    return (
        <>
            <PageSection hasBodyWrapper={false}>
                <Breadcrumb>
                    <BreadcrumbItem>
                        <Link to={vulnerabilitiesPrototypeComponentsPath}>Components</Link>
                    </BreadcrumbItem>
                    <BreadcrumbItem isActive>{data?.name ?? decodedName}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>

            <PageSection hasBodyWrapper={false}>
                {loading && (
                    <Bullseye>
                        <Spinner />
                    </Bullseye>
                )}

                {error && <p>Error loading component: {error.message}</p>}

                {!loading && !error && data && (
                    <>
                        <Flex>
                            <FlexItem>
                                <Title headingLevel="h1">{data.name}</Title>
                            </FlexItem>
                        </Flex>
                        <Flex spaceItems={{ default: 'spaceItemsMd' }}>
                            <FlexItem>
                                <Label color="blue">CVEs: {totalCveCount}</Label>
                            </FlexItem>
                            <FlexItem>
                                <Label color="grey">Images: {totalImageCount}</Label>
                            </FlexItem>
                            <FlexItem>
                                <Label color="grey">Versions: {versions.length}</Label>
                            </FlexItem>
                        </Flex>
                    </>
                )}
            </PageSection>

            <PageSection hasBodyWrapper={false}>
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
            </PageSection>
        </>
    );
}

export default ComponentDetailPage;
