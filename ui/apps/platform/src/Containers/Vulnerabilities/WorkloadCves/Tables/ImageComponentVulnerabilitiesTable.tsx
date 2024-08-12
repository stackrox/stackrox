import React from 'react';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { gql } from '@apollo/client';

import useTableSort from 'hooks/patternfly/useTableSort';
import {
    ImageComponentVulnerability,
    ImageMetadataContext,
    flattenImageComponentVulns,
    imageMetadataContextFragment,
    sortTableData,
} from './table.utils';
import FixedByVersion from '../components/FixedByVersion';
import DockerfileLayer from '../components/DockerfileLayer';
import ComponentLocation from '../components/ComponentLocation';

export { imageMetadataContextFragment };
export type { ImageMetadataContext, ImageComponentVulnerability };

export const imageComponentVulnerabilitiesFragment = gql`
    fragment ImageComponentVulnerabilities on ImageComponent {
        name
        version
        location
        source
        layerIndex
        imageVulnerabilities(query: $query) {
            severity
            fixedByVersion
            pendingExceptionCount: exceptionCount(requestStatus: $statusesForExceptionCount)
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
        <Table
            className="pf-v5-u-p-md"
            style={{
                border: '1px solid var(--pf-v5-c-table--BorderColor)',
            }}
            borders={false}
        >
            <Thead noWrap>
                <Tr>
                    <Th sort={getSortParams('Component')}>Component</Th>
                    <Th>Version</Th>
                    <Th>CVE Fixed in</Th>
                    <Th>Source</Th>
                    <Th>Location</Th>
                </Tr>
            </Thead>
            {sortedComponentVulns.map((componentVuln, index) => {
                const { image, name, version, fixedByVersion, location, source, layer } =
                    componentVuln;
                // No border on the last row
                const style =
                    index !== componentVulns.length - 1
                        ? { borderBottom: '1px solid var(--pf-v5-c-table--BorderColor)' }
                        : {};

                return (
                    <Tbody key={`${image.id}:${name}:${version}`} style={style}>
                        <Tr>
                            <Td>{name}</Td>
                            <Td>{version}</Td>
                            <Td modifier="nowrap">
                                <FixedByVersion fixedByVersion={fixedByVersion} />
                            </Td>
                            <Td>{source}</Td>
                            <Td>
                                <ComponentLocation location={location} source={source} />
                            </Td>
                        </Tr>
                        <Tr>
                            <Td colSpan={5} className="pf-v5-u-pt-0">
                                <DockerfileLayer layer={layer} />
                            </Td>
                        </Tr>
                    </Tbody>
                );
            })}
        </Table>
    );
}

export default ImageComponentVulnerabilitiesTable;
