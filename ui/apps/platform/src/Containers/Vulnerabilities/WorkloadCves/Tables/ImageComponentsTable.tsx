import React from 'react';
import { CodeBlock, CodeBlockCode } from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import { ImageVulnerabilityComponent } from '../hooks/useImageVulnerabilities';
import { ImageDetailsResponse } from '../hooks/useImageDetails';

export type ImageComponentsTableProps = {
    image: ImageDetailsResponse['image'] | undefined;
    imageComponents: ImageVulnerabilityComponent[];
};

function ImageComponentsTable({ image, imageComponents }: ImageComponentsTableProps) {
    const layers = image?.metadata?.v1?.layers ?? [];
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
            {imageComponents.map(({ id, name, version, fixedIn, location, layerIndex }) => {
                let dockerfileText = `* Dockerfile layer information is not available *`;

                if (layerIndex !== null) {
                    const layer = layers[layerIndex];
                    if (layer) {
                        dockerfileText = `${layer.instruction} ${layer.value}`;
                    }
                }
                return (
                    <Tbody
                        key={id}
                        style={{
                            borderBottom: '1px solid var(--pf-c-table--BorderColor)',
                        }}
                    >
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
