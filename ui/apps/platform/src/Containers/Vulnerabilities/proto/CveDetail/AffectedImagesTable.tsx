import { Bullseye, Label } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

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
 * Displays a table of affected images for a given CVE.
 */
function AffectedImagesTable({ images }: AffectedImagesTableProps) {
    return (
        <Table aria-label="Affected images" variant="compact">
            <Thead>
                <Tr>
                    <Th>Image ID</Th>
                    <Th>Components</Th>
                    <Th>Severity</Th>
                    <Th>Fixable</Th>
                </Tr>
            </Thead>
            <Tbody>
                {images.map((img) => (
                    <Tr key={img.imageId}>
                        <Td dataLabel="Image ID">
                            <span title={img.imageId}>
                                {truncateImageId(img.imageId)}
                            </span>
                        </Td>
                        <Td dataLabel="Components">{img.componentCount}</Td>
                        <Td dataLabel="Severity">
                            <Label color={severityColor(img.severity)}>
                                {severityNames[img.severity] ?? 'Unknown'}
                            </Label>
                        </Td>
                        <Td dataLabel="Fixable">
                            {img.fixable ? 'Yes' : 'No'}
                        </Td>
                    </Tr>
                ))}
                {images.length === 0 && (
                    <Tr>
                        <Td colSpan={4}>
                            <Bullseye>No affected images found</Bullseye>
                        </Td>
                    </Tr>
                )}
            </Tbody>
        </Table>
    );
}

export default AffectedImagesTable;
