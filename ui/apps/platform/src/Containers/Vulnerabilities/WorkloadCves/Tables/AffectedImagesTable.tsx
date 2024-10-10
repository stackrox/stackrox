import React from 'react';
import { gql } from '@apollo/client';
import { ExpandableRowContent, Table, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import useFeatureFlags from 'hooks/useFeatureFlags';
import useSet from 'hooks/useSet';
import { UseURLSortResult } from 'hooks/useURLSort';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import { VulnerabilityState } from 'types/cve.proto';
import { DynamicColumnIcon } from 'Components/DynamicIcon';
import CvssFormatted from 'Components/CvssFormatted';
import DateDistance from 'Components/DateDistance';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { TableUIState } from 'utils/getTableUIState';
import ExpandRowTh from 'Components/ExpandRowTh';
import { isNonEmptyArray } from 'utils/type.utils';
import {
    getIsSomeVulnerabilityFixable,
    getHighestCvssScore,
    getHighestNvdCvssScore,
    getHighestVulnerabilitySeverity,
    getEarliestDiscoveredAtTime,
} from '../../utils/vulnerabilityUtils';
import ImageNameLink from '../components/ImageNameLink';

import ImageComponentVulnerabilitiesTable, {
    ImageComponentVulnerability,
    imageComponentVulnerabilitiesFragment,
    imageMetadataContextFragment,
} from './ImageComponentVulnerabilitiesTable';
import { WatchStatus } from '../../types';
import PendingExceptionLabelLayout from '../components/PendingExceptionLabelLayout';

export type ImageForCve = {
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
    operatingSystem: string;
    watchStatus: WatchStatus;
    imageComponents: (ImageComponentVulnerability & {
        imageVulnerabilities: (ImageComponentVulnerability['imageVulnerabilities'][number] & {
            discoveredAtImage: string;
            cvss: number;
            scoreVersion: string;
            nvdCvss: number;
            nvdScoreVersion: string; // for example, V3 or UNKNOWN_VERSION
        })[];
    })[];
};

export const imagesForCveFragment = gql`
    ${imageMetadataContextFragment}
    ${imageComponentVulnerabilitiesFragment}
    fragment ImagesForCVE on Image {
        ...ImageMetadataContext

        operatingSystem
        watchStatus
        imageComponents(query: $query) {
            imageVulnerabilities(query: $query) {
                discoveredAtImage
                cvss
                scoreVersion
                nvdCvss
                nvdScoreVersion
            }
            ...ImageComponentVulnerabilities
        }
    }
`;

export type AffectedImagesTableProps = {
    tableState: TableUIState<ImageForCve>;
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
    cve: string;
    vulnerabilityState: VulnerabilityState | undefined; // TODO Make this required when the ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL feature flag is removed
    onClearFilters: () => void;
};

function AffectedImagesTable({
    tableState,
    getSortParams,
    isFiltered,
    cve,
    vulnerabilityState,
    onClearFilters,
}: AffectedImagesTableProps) {
    const expandedRowSet = useSet<string>();

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isNvdCvssEnabled = isFeatureFlagEnabled('ROX_NVD_CVSS_UI');

    return (
        <Table variant="compact">
            <Thead noWrap>
                <Tr>
                    <ExpandRowTh />
                    <Th sort={getSortParams('Image')}>Image</Th>
                    <Th sort={getSortParams('Severity')}>CVE severity</Th>
                    <Th>
                        CVE status
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th>CVSS</Th>
                    {isNvdCvssEnabled && <Th>NVD CVSS</Th>}
                    <Th sort={getSortParams('Operating System')}>Operating system</Th>
                    <Th>
                        Affected components
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th>First discovered</Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={8}
                emptyProps={{ message: 'No images were found that are affected by this CVE' }}
                filteredEmptyProps={{ onClearFilters }}
                renderer={({ data }) =>
                    data.map((image, rowIndex) => {
                        const { id, name, operatingSystem, imageComponents } = image;
                        const vulnerabilities = imageComponents.flatMap(
                            (imageComponent) => imageComponent.imageVulnerabilities
                        );
                        const topSeverity = getHighestVulnerabilitySeverity(vulnerabilities);
                        const isFixableInImage = getIsSomeVulnerabilityFixable(vulnerabilities);
                        const { cvss, scoreVersion } = getHighestCvssScore(vulnerabilities);
                        const { nvdCvss, nvdScoreVersion } =
                            getHighestNvdCvssScore(vulnerabilities);
                        const hasPendingException = imageComponents.some((imageComponent) =>
                            imageComponent.imageVulnerabilities.some(
                                (imageVulnerability) => imageVulnerability.pendingExceptionCount > 0
                            )
                        );

                        const isExpanded = expandedRowSet.has(id);

                        return (
                            <Tbody key={id} isExpanded={isExpanded}>
                                <Tr>
                                    <Td
                                        expand={{
                                            rowIndex,
                                            isExpanded,
                                            onToggle: () => expandedRowSet.toggle(id),
                                        }}
                                    />
                                    <Td dataLabel="Image">
                                        {name ? (
                                            <PendingExceptionLabelLayout
                                                hasPendingException={hasPendingException}
                                                cve={cve}
                                                vulnerabilityState={vulnerabilityState}
                                            >
                                                <ImageNameLink name={name} id={id} />
                                            </PendingExceptionLabelLayout>
                                        ) : (
                                            'Image name not available'
                                        )}
                                    </Td>
                                    <Td dataLabel="CVE severity" modifier="nowrap">
                                        <VulnerabilitySeverityIconText severity={topSeverity} />
                                    </Td>
                                    <Td dataLabel="CVE status" modifier="nowrap">
                                        <VulnerabilityFixableIconText
                                            isFixable={isFixableInImage}
                                        />
                                    </Td>
                                    <Td dataLabel="CVSS" modifier="nowrap">
                                        <CvssFormatted cvss={cvss} scoreVersion={scoreVersion} />
                                    </Td>
                                    {isNvdCvssEnabled && (
                                        <Td dataLabel="NVD CVSS" modifier="nowrap">
                                            <CvssFormatted
                                                cvss={nvdCvss}
                                                scoreVersion={nvdScoreVersion}
                                            />
                                        </Td>
                                    )}
                                    <Td dataLabel="Operating system">{operatingSystem}</Td>
                                    <Td dataLabel="Affected components">
                                        {imageComponents.length === 1
                                            ? imageComponents[0].name
                                            : `${imageComponents.length} components`}
                                    </Td>
                                    <Td dataLabel="First discovered">
                                        {isNonEmptyArray(vulnerabilities) ? (
                                            <DateDistance
                                                date={getEarliestDiscoveredAtTime(vulnerabilities)}
                                            />
                                        ) : (
                                            'Not available'
                                        )}
                                    </Td>
                                </Tr>
                                <Tr isExpanded={isExpanded}>
                                    <Td />
                                    <Td colSpan={7}>
                                        <ExpandableRowContent>
                                            <ImageComponentVulnerabilitiesTable
                                                imageMetadataContext={image}
                                                componentVulnerabilities={image.imageComponents}
                                            />
                                        </ExpandableRowContent>
                                    </Td>
                                </Tr>
                            </Tbody>
                        );
                    })
                }
            />
        </Table>
    );
}

export default AffectedImagesTable;
