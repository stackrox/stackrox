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
import { TABLE_HEADER_STYLE, TABLE_CELL_STYLE } from '../utils/tableDefaults';

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
            <Thead style={{ borderBottom: '2px solid var(--pf-global--BorderColor--100)' }}>
                <Tr>
                    <Th screenReaderText="Row expansion" style={TABLE_HEADER_STYLE} />
                    <Th style={TABLE_HEADER_STYLE}>Image</Th>
                    <Th style={TABLE_HEADER_STYLE}>Components</Th>
                    <Th style={TABLE_HEADER_STYLE}>Severity</Th>
                    <Th style={TABLE_HEADER_STYLE}>Fixable</Th>
                </Tr>
            </Thead>
            {images.map((img, rowIndex) => {
                const isExpanded = expandedImages.has(img.imageId);
                const imageLink = `/main/vulnerabilities/prototype/images/${encodeURIComponent(img.imageId)}`;
                return (
                    <Tbody key={img.imageId} isExpanded={isExpanded}>
                        <Tr>
                            <Td
                                expand={{
                                    rowIndex,
                                    isExpanded,
                                    onToggle: () => toggleExpand(img.imageId),
                                }}
                                style={TABLE_CELL_STYLE}
                            />
                            <Td dataLabel="Image" style={TABLE_CELL_STYLE}>
                                <Link to={imageLink} title={img.imageId}>
                                    {displayImageName(img)}
                                </Link>
                            </Td>
                            <Td dataLabel="Components" style={TABLE_CELL_STYLE}>
                                {img.componentCount}
                            </Td>
                            <Td dataLabel="Severity" style={TABLE_CELL_STYLE}>
                                <Label color={severityColor(img.severity)}>
                                    {severityNames[img.severity] ?? 'Unknown'}
                                </Label>
                            </Td>
                            <Td dataLabel="Fixable" style={TABLE_CELL_STYLE}>
                                {img.fixable ? 'Yes' : 'No'}
                            </Td>
                        </Tr>
                        <Tr isExpanded={isExpanded}>
                            <Td colSpan={columnCount} style={TABLE_CELL_STYLE}>
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
                                                    <Th style={TABLE_HEADER_STYLE}>Component</Th>
                                                    <Th style={TABLE_HEADER_STYLE}>Version</Th>
                                                    <Th style={TABLE_HEADER_STYLE}>Source</Th>
                                                    <Th style={TABLE_HEADER_STYLE}>Fixed By</Th>
                                                    <Th style={TABLE_HEADER_STYLE}>Advisories</Th>
                                                </Tr>
                                            </Thead>
                                            <Tbody>
                                                {img.components.map(
                                                    (comp, compIdx) => (
                                                        <Tr
                                                            key={`${comp.name}-${comp.version}-${compIdx}`}
                                                        >
                                                            <Td dataLabel="Component" style={TABLE_CELL_STYLE}>
                                                                {comp.name}
                                                            </Td>
                                                            <Td dataLabel="Version" style={TABLE_CELL_STYLE}>
                                                                {comp.version}
                                                            </Td>
                                                            <Td dataLabel="Source" style={TABLE_CELL_STYLE}>
                                                                {comp.source}
                                                            </Td>
                                                            <Td dataLabel="Fixed By" style={TABLE_CELL_STYLE}>
                                                                {comp.fixedBy ||
                                                                    '-'}
                                                            </Td>
                                                            <Td dataLabel="Advisories" style={TABLE_CELL_STYLE}>
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
                        <Td colSpan={columnCount} style={TABLE_CELL_STYLE}>
                            <Bullseye>No affected images found</Bullseye>
                        </Td>
                    </Tr>
                </Tbody>
            )}
        </Table>
    );
}

export default AffectedImagesTable;
