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
import {
    IMAGE_DIGEST_WIDTH,
    COUNT_WIDTH,
    DATE_WIDTH,
    TABLE_HEADER_STYLE,
    TABLE_CELL_STYLE,
    formatDate,
} from '../utils/tableDefaults';
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
                    <Thead isStickyHeader style={{ borderBottom: '2px solid var(--pf-v5-global--BorderColor--100)' }}>
                        <Tr>
                            <Th style={TABLE_HEADER_STYLE}>Image</Th>
                            <Th style={TABLE_HEADER_STYLE}>OS</Th>
                            <Th {...getThSortProps(2)} style={{ ...TABLE_HEADER_STYLE, width: `${COUNT_WIDTH}px` }} info={{ tooltip: 'CVE counts by severity: Critical, Important, Moderate, Low' }}>CVEs</Th>
                            <Th {...getThSortProps(3)} style={TABLE_HEADER_STYLE}>Components</Th>
                            <Th {...getThSortProps(4)} style={TABLE_HEADER_STYLE}>Top Severity</Th>
                            <Th style={TABLE_HEADER_STYLE}>Fixable</Th>
                            <Th style={{ ...TABLE_HEADER_STYLE, width: `${DATE_WIDTH}px` }}>Scan Time</Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {images.map((image) => (
                            <Tr key={image.imageId}>
                                <Td dataLabel="Image" style={TABLE_CELL_STYLE}>
                                    <Link
                                        to={`${vulnerabilitiesPrototypePath}/images/${encodeURIComponent(image.imageId)}`}
                                        style={{
                                            maxWidth: `${IMAGE_DIGEST_WIDTH}px`,
                                            overflow: 'hidden',
                                            textOverflow: 'ellipsis',
                                            whiteSpace: 'nowrap',
                                            display: 'inline-block',
                                        }}
                                        title={imageDisplayName(image)}
                                    >
                                        {imageDisplayName(image)}
                                    </Link>
                                </Td>
                                <Td dataLabel="OS" style={TABLE_CELL_STYLE}>{image.imageOS || '-'}</Td>
                                <Td dataLabel="CVEs" style={TABLE_CELL_STYLE}>
                                    <SeverityBreakdown image={image} />
                                </Td>
                                <Td dataLabel="Components" style={TABLE_CELL_STYLE}>{image.componentCount}</Td>
                                <Td dataLabel="Top Severity" style={TABLE_CELL_STYLE}>
                                    <Label color={severityColor(image.topSeverity)}>
                                        {severityLabel(image.topSeverity)}
                                    </Label>
                                </Td>
                                <Td dataLabel="Fixable" style={TABLE_CELL_STYLE}>
                                    {image.fixable ? 'Yes' : 'No'}
                                </Td>
                                <Td dataLabel="Scan Time" style={TABLE_CELL_STYLE}>
                                    {formatDate(image.scanTime)}
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
