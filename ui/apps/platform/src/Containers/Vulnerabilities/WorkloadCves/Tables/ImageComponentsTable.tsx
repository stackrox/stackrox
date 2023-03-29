import React from 'react';
import { CodeBlock, CodeBlockCode } from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import { ImageMetadataLayer, ImageVulnerabilityComponent } from '../hooks/useImageVulnerabilities';

export type ImageComponentsTableProps = {
    layers: ImageMetadataLayer[];
    imageComponents: ImageVulnerabilityComponent[];
};

function ImageComponentsTable({ layers, imageComponents }: ImageComponentsTableProps) {
    return (
        <TableComposable borders={false}>
            <Thead>
                <Tr>
                    <Th>Component</Th>
                    <Th>Version</Th>
                    <Th>Fixed in</Th>
                    <Th>Location</Th>
                </Tr>
            </Thead>
            {imageComponents.map(({ id, name, version, fixedIn, location, layerIndex }, index) => {
                let dockerfileText = `* Dockerfile layer information is not available *`;

                if (layerIndex !== null) {
                    const layer = layers[layerIndex];
                    if (layer) {
                        dockerfileText = `${layer.instruction} ${layer.value}`;
                    }
                }

                // No border on the last row
                const style =
                    index !== imageComponents.length - 1
                        ? { borderBottom: '1px solid var(--pf-c-table--BorderColor)' }
                        : {};

                return (
                    <Tbody key={id} style={style}>
                        <Tr>
                            <Td>{name}</Td>
                            <Td>{version}</Td>
                            <Td>{fixedIn || 'TODO - why empty'}</Td>
                            <Td>{location || 'TODO - why empty'}</Td>
                        </Tr>
                        <Tr>
                            <Td colSpan={4} className="pf-u-pt-0">
                                <CodeBlock>
                                    <CodeBlockCode>{dockerfileText}</CodeBlockCode>
                                </CodeBlock>
                            </Td>
                        </Tr>
                    </Tbody>
                );
            })}
        </TableComposable>
    );
}

export default ImageComponentsTable;
