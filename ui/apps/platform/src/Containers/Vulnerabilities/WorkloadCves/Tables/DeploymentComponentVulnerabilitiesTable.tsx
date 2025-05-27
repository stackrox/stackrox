import React from 'react';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { gql } from '@apollo/client';

import useFeatureFlags from 'hooks/useFeatureFlags';
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

import AdvisoryLinkOrText from './AdvisoryLinkOrText';

export { imageMetadataContextFragment };
export type { ImageMetadataContext, DeploymentComponentVulnerability };

// After release, replace temporary function
// with deploymentComponentVulnerabilitiesFragment
// that has unconditional advisory property.
export function convertToFlatDeploymentComponentVulnerabilitiesFragment(
    isFlattenCveDataEnabled: boolean // ROX_FLATTEN_CVE_DATA
) {
    return gql`
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
                ${isFlattenCveDataEnabled ? 'advisory { name, link }' : ''}
                discoveredAtImage
                publishedOn
                pendingExceptionCount: exceptionCount(requestStatus: $statusesForExceptionCount)
            }
        }
    `;
}

const sortFields = ['Image', 'Component'];
const defaultSortOption = { field: 'Image', direction: 'asc' } as const;

export type DeploymentComponentVulnerabilitiesTableProps = {
    /** The images and associated component vulnerability data to display in the table */
    images: {
        imageMetadataContext: ImageMetadataContext;
        componentVulnerabilities: DeploymentComponentVulnerability[];
    }[];
    cve: string;
    vulnerabilityState: VulnerabilityState;
};

function DeploymentComponentVulnerabilitiesTable({
    images,
    cve,
    vulnerabilityState,
}: DeploymentComponentVulnerabilitiesTableProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isAdvisoryColumnEnabled =
        isFeatureFlagEnabled('ROX_SCANNER_V4') &&
        isFeatureFlagEnabled('ROX_FLATTEN_CVE_DATA') &&
        isFeatureFlagEnabled('ROX_CVE_ADVISORY_SEPARATION');

    const { sortOption, getSortParams } = useTableSort({ sortFields, defaultSortOption });
    const componentVulns = images.flatMap(({ imageMetadataContext, componentVulnerabilities }) =>
        flattenDeploymentComponentVulns(imageMetadataContext, componentVulnerabilities)
    );
    const sortedComponentVulns = sortTableData(componentVulns, sortOption);

    return (
        <Table
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
                    {isAdvisoryColumnEnabled && <Th>Advisory</Th>}
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
                    advisory,
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
                            <Td dataLabel="Image">
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
                            <Td dataLabel="CVE severity" modifier="nowrap">
                                <VulnerabilitySeverityIconText severity={severity} />
                            </Td>
                            <Td dataLabel="CVSS" modifier="nowrap">
                                <CvssFormatted cvss={cvss} scoreVersion={scoreVersion} />
                            </Td>
                            <Td dataLabel="Component">{name}</Td>
                            <Td dataLabel="Version">{version}</Td>
                            <Td dataLabel="CVE fixed in" modifier="nowrap">
                                <FixedByVersion fixedByVersion={fixedByVersion} />
                            </Td>
                            {isAdvisoryColumnEnabled && (
                                <Td dataLabel="Advisory" modifier="nowrap">
                                    <AdvisoryLinkOrText advisory={advisory} />
                                </Td>
                            )}
                            <Td dataLabel="Source">{source}</Td>
                            <Td dataLabel="Location">
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
