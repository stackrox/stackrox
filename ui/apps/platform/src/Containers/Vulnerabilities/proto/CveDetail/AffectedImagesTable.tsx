import { useState } from 'react';

import { Bullseye, Label } from '@patternfly/react-core';
import {
    ExpandableRowContent,
    Table,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';

import type { ProtoImage } from './useCveDetail';

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

/**
 * Truncates a sha256 image ID for display (e.g. "sha256:abc123..." -> "sha256:abc123..").
 */
function truncateImageId(imageId: string): string {
    if (imageId.startsWith('sha256:') && imageId.length > 19) {
        return `${imageId.slice(0, 19)}...`;
    }
    if (imageId.length > 12) {
        return `${imageId.slice(0, 12)}...`;
    }
    return imageId;
}

type AffectedImagesTableProps = {
    images: ProtoImage[];
};

/**
 * Displays a table of affected images for a given CVE, with expandable rows
 * showing the components affected within each image.
 */
function AffectedImagesTable({ images }: AffectedImagesTableProps) {
    const [expandedImages, setExpandedImages] = useState<Set<string>>(
        new Set()
    );

    function toggleExpand(imageId: string) {
        setExpandedImages((prev) => {
            const next = new Set(prev);
            if (next.has(imageId)) {
                next.delete(imageId);
            } else {
                next.add(imageId);
            }
            return next;
        });
    }

    const columnCount = 5; // toggle + image ID + components + severity + fixable

    return (
        <Table aria-label="Affected images" variant="compact">
            <Thead>
                <Tr>
                    <Th screenReaderText="Row expansion" />
                    <Th>Image ID</Th>
                    <Th>Components</Th>
                    <Th>Severity</Th>
                    <Th>Fixable</Th>
                </Tr>
            </Thead>
            {images.map((img, rowIndex) => {
                const isExpanded = expandedImages.has(img.imageId);
                const imageDetailUrl = `/main/vulnerabilities/workload-cves/images/${img.imageId}`;
                return (
                    <Tbody key={img.imageId} isExpanded={isExpanded}>
                        <Tr>
                            <Td
                                expand={{
                                    rowIndex,
                                    isExpanded,
                                    onToggle: () => toggleExpand(img.imageId),
                                }}
                            />
                            <Td dataLabel="Image ID">
                                <a
                                    href={imageDetailUrl}
                                    title={img.imageId}
                                >
                                    {truncateImageId(img.imageId)}
                                </a>
                            </Td>
                            <Td dataLabel="Components">
                                {img.componentCount}
                            </Td>
                            <Td dataLabel="Severity">
                                <Label color={severityColor(img.severity)}>
                                    {severityNames[img.severity] ?? 'Unknown'}
                                </Label>
                            </Td>
                            <Td dataLabel="Fixable">
                                {img.fixable ? 'Yes' : 'No'}
                            </Td>
                        </Tr>
                        <Tr isExpanded={isExpanded}>
                            <Td colSpan={columnCount}>
                                <ExpandableRowContent>
                                    {img.components &&
                                    img.components.length > 0 ? (
                                        <Table
                                            aria-label={`Components for ${truncateImageId(img.imageId)}`}
                                            variant="compact"
                                            borders={false}
                                        >
                                            <Thead>
                                                <Tr>
                                                    <Th>Component</Th>
                                                    <Th>Version</Th>
                                                    <Th>Source</Th>
                                                    <Th>Fixed By</Th>
                                                </Tr>
                                            </Thead>
                                            <Tbody>
                                                {img.components.map(
                                                    (comp, compIdx) => (
                                                        <Tr
                                                            key={`${comp.name}-${comp.version}-${compIdx}`}
                                                        >
                                                            <Td dataLabel="Component">
                                                                {comp.name}
                                                            </Td>
                                                            <Td dataLabel="Version">
                                                                {comp.version}
                                                            </Td>
                                                            <Td dataLabel="Source">
                                                                {comp.source}
                                                            </Td>
                                                            <Td dataLabel="Fixed By">
                                                                {comp.fixedBy ||
                                                                    '-'}
                                                            </Td>
                                                        </Tr>
                                                    )
                                                )}
                                            </Tbody>
                                        </Table>
                                    ) : (
                                        <Bullseye>
                                            No component details available
                                        </Bullseye>
                                    )}
                                </ExpandableRowContent>
                            </Td>
                        </Tr>
                    </Tbody>
                );
            })}
            {images.length === 0 && (
                <Tbody>
                    <Tr>
                        <Td colSpan={columnCount}>
                            <Bullseye>No affected images found</Bullseye>
                        </Td>
                    </Tr>
                </Tbody>
            )}
        </Table>
    );
}

export default AffectedImagesTable;
