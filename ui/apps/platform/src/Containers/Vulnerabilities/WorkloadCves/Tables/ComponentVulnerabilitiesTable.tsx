import React from 'react';
import { CodeBlock, CodeBlockCode, Flex } from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { gql } from '@apollo/client';
import sortBy from 'lodash/sortBy';

import { isVulnerabilitySeverity } from 'types/cve.proto';
import { ApiSortOption } from 'types/search';
import { NotFixableIcon } from 'Components/PatternFly/FixabilityIcons';
import useTableSort from 'hooks/patternfly/useTableSort';

export type ImageMetadataContext = {
    id: string;
    name: {
        registry: string;
        remote: string;
        tag: string;
    } | null;
    metadata: {
        v1: {
            layers: {
                instruction: string;
                value: string;
            }[];
        } | null;
    } | null;
};

export const imageMetadataContextFragment = gql`
    fragment ImageMetadataContext on Image {
        id
        name {
            registry
            remote
            tag
        }
        metadata {
            v1 {
                layers {
                    instruction
                    value
                }
            }
        }
    }
`;

export type ComponentVulnerabilities = {
    imageComponents: {
        name: string;
        version: string;
        location: string;
        layerIndex: number | null;
        imageVulnerabilities: {
            severity: string;
            fixedByVersion: string;
        }[];
    }[];
};

export const componentVulnerabilitiesFragment = gql`
    fragment ComponentVulnerabilities on Image {
        imageComponents(query: $vulnCveQuery) {
            name
            version
            location
            layerIndex
            imageVulnerabilities(query: $vulnCveQuery) {
                severity
                fixedByVersion
            }
        }
    }
`;

type TableDataRow = {
    image: {
        id: string;
        name: {
            remote: string;
            registry: string;
        } | null;
    };
    name: string;
    severity: string;
    version: string;
    fixedIn: string;
    location: string;
    layer: {
        line: number;
        instruction: string;
        value: string;
    } | null;
};

/**
 * Given an image and its nested components and vulnerabilities, flatten the data into a single
 * level for display in a table. Note that this function assumes that the vulnerabilities array
 * for each component only has one element, which is the case when the query is filtered by CVE ID.
 *
 * @param imageContext The image context to use for the table rows
 * @param componentVulnerabilities The nested component -> vulnerabilities data for the image
 *
 * @returns The flattened table data
 */
function flattenImageComponentVulns(
    imageContext: ImageMetadataContext,
    componentVulnerabilities: ComponentVulnerabilities['imageComponents']
): TableDataRow[] {
    const rows: TableDataRow[] = [];
    componentVulnerabilities.forEach((component) => {
        const image = imageContext;
        const layers = imageContext.metadata?.v1?.layers ?? [];
        const { name, location, version, layerIndex } = component;

        let layer: TableDataRow['layer'] = null;

        if (layerIndex !== null) {
            const targetLayer = layers[layerIndex];
            if (targetLayer) {
                layer = {
                    line: layerIndex + 1,
                    instruction: targetLayer.instruction,
                    value: targetLayer.value,
                };
            }
        }

        // This imageVulnerabilities array should always only have one element
        // because we are filtering by CVE ID to get the data
        const vulnerability = component.imageVulnerabilities[0];
        const severity =
            vulnerability && isVulnerabilitySeverity(vulnerability.severity)
                ? vulnerability.severity
                : 'UNKNOWN_VULNERABILITY_SEVERITY';

        const fixedIn = vulnerability?.fixedByVersion ?? 'N/A';

        rows.push({ image, name, severity, version, fixedIn, location, layer });
    });

    return rows;
}

function sortTableData(tableData: TableDataRow[], sortOption: ApiSortOption): TableDataRow[] {
    const sortedRows = sortBy(tableData, (row) => {
        switch (sortOption.field) {
            case 'Image':
                return row.image.name?.remote ?? '';
            case 'Component':
                return row.name;
            default:
                return '';
        }
    });

    if (sortOption.reversed) {
        sortedRows.reverse();
    }
    return sortedRows;
}

const sortFields = ['Image', 'Component'];

export type ImageComponentVulnerabilitiesTableProps = {
    /** Whether to show the image column */
    showImage?: boolean;
    /** The images and associated component vulnerability data to display in the table */
    images: {
        context: ImageMetadataContext;
        componentVulnerabilities: ComponentVulnerabilities['imageComponents'];
    }[];
};

function ComponentVulnerabilitiesTable({
    showImage = false,
    images,
}: ImageComponentVulnerabilitiesTableProps) {
    const defaultSortOption = {
        field: showImage ? 'Image' : 'Component',
        direction: 'asc',
    } as const;
    const { sortOption, getSortParams } = useTableSort({
        sortFields,
        defaultSortOption,
    });
    const componentVulns = images.flatMap(({ context, componentVulnerabilities }) =>
        flattenImageComponentVulns(context, componentVulnerabilities)
    );
    const sortedComponentVulns = sortTableData(componentVulns, sortOption);
    return (
        <TableComposable
            className="pf-u-p-md"
            style={{
                border: '1px solid var(--pf-c-table--BorderColor)',
            }}
            borders={false}
        >
            <Thead>
                <Tr>
                    {showImage && <Th sort={getSortParams('Image')}>Image</Th>}
                    <Th sort={getSortParams('Component')}>Component</Th>
                    <Th>Version</Th>
                    <Th>Fixed in</Th>
                    <Th>Location</Th>
                </Tr>
            </Thead>
            {sortedComponentVulns.map(({ image, name, version, fixedIn, location, layer }) => {
                return (
                    <Tbody
                        key={`${image.id}:${name}`}
                        style={{
                            borderBottom: '1px solid var(--pf-c-table--BorderColor)',
                        }}
                    >
                        <Tr>
                            {showImage && <Td>{image?.name?.remote ?? ''} TODO - Extract</Td>}
                            <Td>{name}</Td>
                            <Td>{version}</Td>
                            <Td>
                                {fixedIn || (
                                    <Flex
                                        alignItems={{ default: 'alignItemsCenter' }}
                                        spaceItems={{ default: 'spaceItemsSm' }}
                                    >
                                        <NotFixableIcon />
                                        <span>Not fixable</span>
                                    </Flex>
                                )}
                            </Td>
                            <Td>{location || 'N/A'}</Td>
                        </Tr>
                        <Tr>
                            <Td colSpan={showImage ? 5 : 4} className="pf-u-pt-0">
                                {layer ? (
                                    <CodeBlock>
                                        <Flex>
                                            <CodeBlockCode
                                                // 120px is a width that looks good with the largest dockerfile instruction: "HEALTHCHECK"
                                                style={{ flexBasis: '120px' }}
                                                className="pf-u-flex-shrink-0"
                                            >
                                                {layer.line} {layer.instruction}
                                            </CodeBlockCode>
                                            <CodeBlockCode className="pf-u-flex-grow-1 pf-u-flex-basis-0">
                                                {layer.instruction} {layer.value}
                                            </CodeBlockCode>
                                        </Flex>
                                    </CodeBlock>
                                ) : (
                                    <CodeBlock>
                                        <CodeBlockCode>
                                            Dockerfile layer information not available
                                        </CodeBlockCode>
                                    </CodeBlock>
                                )}
                            </Td>
                        </Tr>
                    </Tbody>
                );
            })}
        </TableComposable>
    );
}

export default ComponentVulnerabilitiesTable;
