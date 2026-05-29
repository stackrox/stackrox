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
import { Link } from 'react-router-dom-v5-compat';

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

function displayImageName(img: ProtoImage): string {
    if (img.imageName) {
        return img.imageName;
    }
    if (img.imageId.startsWith('sha256:') && img.imageId.length > 19) {
        return `${img.imageId.slice(0, 19)}...`;
    }
    return img.imageId;
}

type AffectedImagesTableProps = {
    images: ProtoImage[];
};

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

    const columnCount = 5;

    return (
        <Table aria-label="Affected images" variant="compact">
            <Thead>
                <Tr>
                    <Th screenReaderText="Row expansion" />
                    <Th>Image</Th>
                    <Th>Components</Th>
                    <Th>Severity</Th>
                    <Th>Fixable</Th>
                </Tr>
            </Thead>
            {images.map((img, rowIndex) => {
                const isExpanded = expandedImages.has(img.imageId);
                const imageLink = img.imageUuid
                    ? `/main/vulnerabilities/workload-cves/images/${img.imageUuid}`
                    : null;
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
                            <Td dataLabel="Image">
                                {imageLink ? (
                                    <Link to={imageLink} title={img.imageId}>
                                        {displayImageName(img)}
                                    </Link>
                                ) : (
                                    <span title={img.imageId}>
                                        {displayImageName(img)}
                                    </span>
                                )}
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
                                            aria-label={`Components for ${displayImageName(img)}`}
                                            variant="compact"
                                            borders={false}
                                        >
                                            <Thead>
                                                <Tr>
                                                    <Th>Component</Th>
                                                    <Th>Version</Th>
                                                    <Th>Source</Th>
                                                    <Th>Fixed By</Th>
                                                    <Th>Advisories</Th>
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
                                                            <Td dataLabel="Advisories">
                                                                {comp.advisories?.join(
                                                                    ', '
                                                                ) || '-'}
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
