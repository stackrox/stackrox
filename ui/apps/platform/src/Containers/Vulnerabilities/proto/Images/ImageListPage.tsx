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

import { vulnerabilitiesPrototypePath } from 'routePaths';

import ProtoNav from '../ProtoNav';
import { usePagination } from '../usePagination';
import { useSort } from '../useSort';
import { useImageList } from './useImageList';
import type { ProtoImageListItem } from './useImageList';

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
 * Renders severity breakdown badges for an image row.
 */
function SeverityBreakdown({ image }: { image: ProtoImageListItem }) {
    const badges: SeverityBadgeProps[] = [
        { label: 'C', count: image.criticalCount, color: 'red' },
        { label: 'I', count: image.importantCount, color: 'orange' },
        { label: 'M', count: image.moderateCount, color: 'blue' },
        { label: 'L', count: image.lowCount, color: 'yellow' },
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

/**
 * Returns a display name for an image: the full name if available, otherwise a truncated SHA.
 */
function imageDisplayName(image: ProtoImageListItem): string {
    if (image.imageName) {
        return image.imageName;
    }
    if (image.imageId.length > 20) {
        return `${image.imageId.substring(0, 16)}...`;
    }
    return image.imageId;
}

/**
 * Formats a scan time string for display.
 */
function formatScanTime(scanTime: string | null): string {
    if (!scanTime) {
        return '-';
    }
    try {
        return new Date(scanTime).toLocaleString();
    } catch {
        return scanTime;
    }
}

// Column keys: non-sortable columns use empty string.
const imageSortColumns = ['', '', 'cveCount', 'componentCount', 'severity', '', ''];

function ImageListPage() {
    const { sortBy, sortDir, getThSortProps } = useSort(imageSortColumns, 4);
    const { page, perPage, offset, onSetPage, onPerPageSelect } = usePagination(20);
    const { data, loading, error } = useImageList(perPage, offset, sortBy, sortDir);

    const images: ProtoImageListItem[] = data?.images ?? [];
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
                                `${images.length} of ${totalCount} Images`}
                        </ToolbarItem>
                    </ToolbarContent>
                </Toolbar>

                {error && (
                    <Bullseye>
                        <p>Error loading images: {error.message}</p>
                    </Bullseye>
                )}

                <Table aria-label="Vuln Management V5 image list" variant="compact">
                    <Thead>
                        <Tr>
                            <Th>Image</Th>
                            <Th>OS</Th>
                            <Th {...getThSortProps(2)} info={{ tooltip: 'CVE counts by severity: Critical, Important, Moderate, Low' }}>CVEs</Th>
                            <Th {...getThSortProps(3)}>Components</Th>
                            <Th {...getThSortProps(4)}>Top Severity</Th>
                            <Th>Fixable</Th>
                            <Th>Scan Time</Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {images.map((image) => (
                            <Tr key={image.imageId}>
                                <Td dataLabel="Image">
                                    <Link
                                        to={`${vulnerabilitiesPrototypePath}/images/${encodeURIComponent(image.imageId)}`}
                                    >
                                        {imageDisplayName(image)}
                                    </Link>
                                </Td>
                                <Td dataLabel="OS">{image.imageOS || '-'}</Td>
                                <Td dataLabel="CVEs">
                                    <SeverityBreakdown image={image} />
                                </Td>
                                <Td dataLabel="Components">{image.componentCount}</Td>
                                <Td dataLabel="Top Severity">
                                    <Label color={severityColor(image.topSeverity)}>
                                        {severityLabel(image.topSeverity)}
                                    </Label>
                                </Td>
                                <Td dataLabel="Fixable">
                                    {image.fixable ? 'Yes' : 'No'}
                                </Td>
                                <Td dataLabel="Scan Time">
                                    {formatScanTime(image.scanTime)}
                                </Td>
                            </Tr>
                        ))}
                        {!loading && images.length === 0 && (
                            <Tr>
                                <Td colSpan={7}>
                                    <Bullseye>No images found</Bullseye>
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

export default ImageListPage;
