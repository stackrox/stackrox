import { Label } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { gql } from '@apollo/client';

import useFeatureFlags from 'hooks/useFeatureFlags';
import useTableSort from 'hooks/useTableSort';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import CvssFormatted from 'Components/CvssFormatted';

import AdvisoryLinkOrText from '../../components/AdvisoryLinkOrText';
import { getOriginDisplayName } from '../../utils/vulnerabilityUtils';
import {
    flattenImageComponentVulns,
    imageMetadataContextFragment,
    sortTableData,
} from './table.utils';
import type { ImageComponentVulnerability, ImageMetadataContext } from './table.utils';
import DockerfileLayer from '../components/DockerfileLayer';
import ComponentLocation from '../components/ComponentLocation';
import FixedByVersion from '../../components/FixedByVersion';

export { imageMetadataContextFragment };
export type { ImageMetadataContext, ImageComponentVulnerability };

export const imageComponentVulnerabilitiesFragment = gql`
    fragment ImageComponentVulnerabilities on ImageComponent {
        name
        version
        location
        source
        layerIndex
        inBaseImageLayer
        imageVulnerabilities(query: $query) {
            severity
            cvss
            scoreVersion
            fixedByVersion
            origin
            advisory {
                name
                link
            }
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
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isAdvisoryColumnEnabled = isFeatureFlagEnabled('ROX_SCANNER_V4');
    const isOriginColumnEnabled = isFeatureFlagEnabled('ROX_SCANNER_V4');
    const isLayerTypeColumnEnabled = isFeatureFlagEnabled('ROX_BASE_IMAGE_DETECTION');

    const colSpanForDockerfileLayer =
        7 +
        (isAdvisoryColumnEnabled ? 1 : 0) +
        (isOriginColumnEnabled ? 1 : 0) +
        (isLayerTypeColumnEnabled ? 1 : 0);

    const { sortOption, getSortParams } = useTableSort({ sortFields, defaultSortOption });
    const componentVulns = flattenImageComponentVulns(
        imageMetadataContext,
        componentVulnerabilities
    );
    const sortedComponentVulns = sortTableData(componentVulns, sortOption);

    return (
        <Table borders={false} variant="compact">
            <Thead noWrap>
                <Tr>
                    <Th sort={getSortParams('Component')}>Component</Th>
                    <Th>Version</Th>
                    <Th>CVE severity</Th>
                    <Th>CVSS</Th>
                    <Th>CVE fixed in</Th>
                    {isAdvisoryColumnEnabled && <Th>Advisory</Th>}
                    <Th>Source</Th>
                    {isOriginColumnEnabled && <Th>CVE origin</Th>}
                    {isLayerTypeColumnEnabled && <Th>Layer type</Th>}
                    <Th>Location</Th>
                </Tr>
            </Thead>
            {sortedComponentVulns.map((componentVuln, index) => {
                const {
                    image,
                    name,
                    version,
                    severity,
                    cvss,
                    scoreVersion,
                    fixedByVersion,
                    origin,
                    advisory,
                    location,
                    source,
                    layer,
                    inBaseImageLayer = false,
                } = componentVuln;
                // No border on the last row
                const style =
                    index !== componentVulns.length - 1
                        ? {
                              borderBlockEnd:
                                  '1px solid var(--pf-v6-c-table__tr--BorderBlockEndColor)',
                          }
                        : {};

                return (
                    <Tbody key={`${image.id}:${name}:${version}`}>
                        <Tr>
                            <Td dataLabel="Component">{name}</Td>
                            <Td dataLabel="Version">{version}</Td>
                            <Td dataLabel="CVE severity" modifier="nowrap">
                                <VulnerabilitySeverityIconText severity={severity} />
                            </Td>
                            <Td dataLabel="CVSS" modifier="nowrap">
                                <CvssFormatted cvss={cvss} scoreVersion={scoreVersion} />
                            </Td>
                            <Td dataLabel="CVE fixed in" modifier="nowrap">
                                <FixedByVersion fixedByVersion={fixedByVersion} />
                            </Td>
                            {isAdvisoryColumnEnabled && (
                                <Td dataLabel="Advisory" modifier="nowrap">
                                    <AdvisoryLinkOrText advisory={advisory} />
                                </Td>
                            )}
                            <Td dataLabel="Source">{source}</Td>
                            {isOriginColumnEnabled && (
                                <Td dataLabel="CVE origin">{getOriginDisplayName(origin)}</Td>
                            )}
                            {isLayerTypeColumnEnabled && (
                                <Td dataLabel="Layer type">
                                    <Label color={inBaseImageLayer ? 'blue' : 'grey'} isCompact>
                                        {inBaseImageLayer ? 'Base image' : 'Application'}
                                    </Label>
                                </Td>
                            )}
                            <Td dataLabel="Location">
                                <ComponentLocation location={location} source={source} />
                            </Td>
                        </Tr>
                        <Tr style={style}>
                            <Td colSpan={colSpanForDockerfileLayer} className="pf-v6-u-pt-0">
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
