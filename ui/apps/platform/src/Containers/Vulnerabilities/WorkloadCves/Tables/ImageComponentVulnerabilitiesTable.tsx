import React from 'react';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { gql } from '@apollo/client';

import useTableSort from 'hooks/patternfly/useTableSort';
import {
    ImageComponentVulnerability,
    ImageMetadataContext,
    flattenImageComponentVulns,
    imageMetadataContextFragment,
    sortTableData,
} from './table.utils';
import FixedByVersionTd from '../components/FixedByVersionTd';
import DockerfileLayerTd from '../components/DockerfileLayerTd';

export { imageMetadataContextFragment };
export type { ImageMetadataContext, ImageComponentVulnerability };

export const imageComponentVulnerabilitiesFragment = gql`
    fragment ImageComponentVulnerabilities on ImageComponent {
        name
        version
        location
        layerIndex
        imageVulnerabilities(query: $query) {
            id
            severity
            fixedByVersion
        }
    }
`;

const sortFields = ['Component'];
const defaultSortOption = { field: 'Component', direction: 'asc' } as const;

export type ImageComponentVulnerabilitiesTableProps = {
    /** The image and associated component vulnerability data to display in the table */
    imageMetadataContext: ImageMetadataContext;
    componentVulnerabilities: ImageComponentVulnerability[];
};

function ImageComponentVulnerabilitiesTable({
    imageMetadataContext,
    componentVulnerabilities,
}: ImageComponentVulnerabilitiesTableProps) {
    const { sortOption, getSortParams } = useTableSort({ sortFields, defaultSortOption });
    const componentVulns = flattenImageComponentVulns(
        imageMetadataContext,
        componentVulnerabilities
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
                    <Th sort={getSortParams('Component')}>Component</Th>
                    <Th>Version</Th>
                    <Th>CVE Fixed in</Th>
                    <Th>Location</Th>
                </Tr>
            </Thead>
            {sortedComponentVulns.map((componentVuln, index) => {
                const { image, name, vulnerabilityId, version, fixedByVersion, location, layer } =
                    componentVuln;
                // No border on the last row
                const style =
                    index !== componentVulns.length - 1
                        ? { borderBottom: '1px solid var(--pf-c-table--BorderColor)' }
                        : {};

                return (
                    <Tbody key={`${image.id}:${name}:${version}:${vulnerabilityId}`} style={style}>
                        <Tr>
                            <Td>{name}</Td>
                            <Td>{version}</Td>
                            <Td>
                                <FixedByVersionTd fixedByVersion={fixedByVersion} />
                            </Td>
                            <Td>{location || 'N/A'}</Td>
                        </Tr>
                        <Tr>
                            <Td colSpan={4} className="pf-u-pt-0">
                                <DockerfileLayerTd layer={layer} />
                            </Td>
                        </Tr>
                    </Tbody>
                );
            })}
        </TableComposable>
    );
}

export default ImageComponentVulnerabilitiesTable;
