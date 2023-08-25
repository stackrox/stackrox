import React from 'react';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { gql } from '@apollo/client';

import useTableSort from 'hooks/patternfly/useTableSort';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import ImageNameTd from '../components/ImageNameTd';
import {
    imageMetadataContextFragment,
    ImageMetadataContext,
    DeploymentComponentVulnerability,
    sortTableData,
    flattenDeploymentComponentVulns,
} from './table.utils';
import FixedByVersionTd from '../components/FixedByVersionTd';
import DockerfileLayerTd from '../components/DockerfileLayerTd';
import ComponentLocationTd from '../components/ComponentLocationTd';
import CvssTd from '../components/CvssTd';

export { imageMetadataContextFragment };
export type { ImageMetadataContext, DeploymentComponentVulnerability };

export const deploymentComponentVulnerabilitiesFragment = gql`
    fragment DeploymentComponentVulnerabilities on ImageComponent {
        name
        version
        location
        source
        layerIndex
        imageVulnerabilities(query: $query) {
            vulnerabilityId: id
            severity
            cvss
            scoreVersion
            fixedByVersion
            discoveredAtImage
        }
    }
`;

const sortFields = ['Image', 'Component'];
const defaultSortOption = { field: 'Image', direction: 'asc' } as const;

export type DeploymentComponentVulnerabilitiesTableProps = {
    /** The images and associated component vulnerability data to display in the table */
    images: {
        imageMetadataContext: ImageMetadataContext;
        componentVulnerabilities: DeploymentComponentVulnerability[];
    }[];
};

function DeploymentComponentVulnerabilitiesTable({
    images,
}: DeploymentComponentVulnerabilitiesTableProps) {
    const { sortOption, getSortParams } = useTableSort({ sortFields, defaultSortOption });
    const componentVulns = images.flatMap(({ imageMetadataContext, componentVulnerabilities }) =>
        flattenDeploymentComponentVulns(imageMetadataContext, componentVulnerabilities)
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
            <Thead noWrap>
                <Tr>
                    <Th sort={getSortParams('Image')}>Image</Th>
                    <Th>CVE severity</Th>
                    <Th>CVSS</Th>
                    <Th sort={getSortParams('Component')}>Component</Th>
                    <Th>Version</Th>
                    <Th>CVE fixed in</Th>
                    <Th>Location</Th>
                </Tr>
            </Thead>
            {sortedComponentVulns.map((componentVuln, index) => {
                const {
                    image,
                    name,
                    vulnerabilityId,
                    severity,
                    version,
                    cvss,
                    scoreVersion,
                    fixedByVersion,
                    location,
                    source,
                    layer,
                } = componentVuln;
                // No border on the last row
                const style =
                    index !== componentVulns.length - 1
                        ? { borderBottom: '1px solid var(--pf-c-table--BorderColor)' }
                        : {};

                return (
                    <Tbody key={`${image.id}:${name}:${version}:${vulnerabilityId}`} style={style}>
                        <Tr>
                            <Td>
                                {image.name ? (
                                    <ImageNameTd name={image.name} id={image.id} />
                                ) : (
                                    'Image name not available'
                                )}
                            </Td>
                            <Td modifier="nowrap">
                                <VulnerabilitySeverityIconText severity={severity} />
                            </Td>
                            <Td modifier="nowrap">
                                <CvssTd cvss={cvss} scoreVersion={scoreVersion} />
                            </Td>
                            <Td>{name}</Td>
                            <Td>{version}</Td>
                            <Td modifier="nowrap">
                                <FixedByVersionTd fixedByVersion={fixedByVersion} />
                            </Td>
                            <Td>
                                <ComponentLocationTd location={location} source={source} />
                            </Td>
                        </Tr>
                        <Tr>
                            <Td colSpan={7} className="pf-u-pt-0">
                                <DockerfileLayerTd layer={layer} />
                            </Td>
                        </Tr>
                    </Tbody>
                );
            })}
        </TableComposable>
    );
}

export default DeploymentComponentVulnerabilitiesTable;
