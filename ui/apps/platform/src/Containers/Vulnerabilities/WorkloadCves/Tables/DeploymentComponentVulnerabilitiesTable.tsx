import React from 'react';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { gql } from '@apollo/client';

import useTableSort from 'hooks/patternfly/useTableSort';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import { VulnerabilityState } from 'types/cve.proto';
import CvssFormatted from 'Components/CvssFormatted';
import ImageNameLink from '../components/ImageNameLink';
import {
    imageMetadataContextFragment,
    ImageMetadataContext,
    DeploymentComponentVulnerability,
    sortTableData,
    flattenDeploymentComponentVulns,
} from './table.utils';
import FixedByVersion from '../components/FixedByVersion';
import DockerfileLayer from '../components/DockerfileLayer';
import ComponentLocation from '../components/ComponentLocation';
import PendingExceptionLabelLayout from '../components/PendingExceptionLabelLayout';

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
            severity
            cvss
            scoreVersion
            fixedByVersion
            discoveredAtImage
            pendingExceptionCount: exceptionCount(requestStatus: $statusesForExceptionCount)
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
    cve: string;
    vulnerabilityState: VulnerabilityState | undefined; // TODO Make this required when the ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL feature flag is removed
};

function DeploymentComponentVulnerabilitiesTable({
    images,
    cve,
    vulnerabilityState,
}: DeploymentComponentVulnerabilitiesTableProps) {
    const { sortOption, getSortParams } = useTableSort({ sortFields, defaultSortOption });
    const componentVulns = images.flatMap(({ imageMetadataContext, componentVulnerabilities }) =>
        flattenDeploymentComponentVulns(imageMetadataContext, componentVulnerabilities)
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
                    <Th sort={getSortParams('Image')}>Image</Th>
                    <Th>CVE severity</Th>
                    <Th>CVSS</Th>
                    <Th sort={getSortParams('Component')}>Component</Th>
                    <Th>Version</Th>
                    <Th>CVE fixed in</Th>
                    <Th>Source</Th>
                    <Th>Location</Th>
                </Tr>
            </Thead>
            {sortedComponentVulns.map((componentVuln, index) => {
                const {
                    image,
                    name,
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
                        ? { borderBottom: '1px solid var(--pf-v5-c-table--BorderColor)' }
                        : {};
                const hasPendingException = componentVulns.some(
                    (vuln) => vuln.pendingExceptionCount > 0
                );

                return (
                    <Tbody key={`${image.id}:${name}:${version}`} style={style}>
                        <Tr>
                            <Td>
                                {image.name ? (
                                    <PendingExceptionLabelLayout
                                        hasPendingException={hasPendingException}
                                        cve={cve}
                                        vulnerabilityState={vulnerabilityState}
                                    >
                                        <ImageNameLink name={image.name} id={image.id} />
                                    </PendingExceptionLabelLayout>
                                ) : (
                                    'Image name not available'
                                )}
                            </Td>
                            <Td modifier="nowrap">
                                <VulnerabilitySeverityIconText severity={severity} />
                            </Td>
                            <Td modifier="nowrap">
                                <CvssFormatted cvss={cvss} scoreVersion={scoreVersion} />
                            </Td>
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
                            <Td colSpan={8} className="pf-v5-u-pt-0">
                                <DockerfileLayer layer={layer} />
                            </Td>
                        </Tr>
                    </Tbody>
                );
            })}
        </Table>
    );
}

export default DeploymentComponentVulnerabilitiesTable;
