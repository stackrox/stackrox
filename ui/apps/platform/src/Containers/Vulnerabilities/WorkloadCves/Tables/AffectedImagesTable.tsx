import React from 'react';
import { gql } from '@apollo/client';
import { ExpandableRowContent, Table, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import useSet from 'hooks/useSet';
import { UseURLSortResult } from 'hooks/useURLSort';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import { VulnerabilityState } from 'types/cve.proto';
import { DynamicColumnIcon } from 'Components/DynamicIcon';
import CvssFormatted from 'Components/CvssFormatted';
import DateDistance from 'Components/DateDistance';
import {
    getIsSomeVulnerabilityFixable,
    getHighestCvssScore,
    getHighestVulnerabilitySeverity,
} from '../../utils/vulnerabilityUtils';
import ImageNameLink from '../components/ImageNameLink';

import ImageComponentVulnerabilitiesTable, {
    ImageComponentVulnerability,
    imageComponentVulnerabilitiesFragment,
    imageMetadataContextFragment,
} from './ImageComponentVulnerabilitiesTable';
import EmptyTableResults from '../components/EmptyTableResults';
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
    scanTime: string | null;
    imageComponents: (ImageComponentVulnerability & {
        imageVulnerabilities: (ImageComponentVulnerability['imageVulnerabilities'][number] & {
            cvss: number;
            scoreVersion: string;
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
        scanTime

        imageComponents(query: $query) {
            imageVulnerabilities(query: $query) {
                cvss
                scoreVersion
            }
            ...ImageComponentVulnerabilities
        }
    }
`;

export type AffectedImagesTableProps = {
    images: ImageForCve[];
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
    cve: string;
    vulnerabilityState: VulnerabilityState | undefined; // TODO Make this required when the ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL feature flag is removed
};

function AffectedImagesTable({
    images,
    getSortParams,
    isFiltered,
    cve,
    vulnerabilityState,
}: AffectedImagesTableProps) {
    const expandedRowSet = useSet<string>();

    return (
        <Table variant="compact">
            <Thead noWrap>
                <Tr>
                    <Th>{/* Header for expanded column */}</Th>
                    <Th sort={getSortParams('Image')}>Image</Th>
                    <Th>CVE severity</Th>
                    <Th>CVSS</Th>
                    <Th>
                        CVE status
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th sort={getSortParams('Operating System')}>Operating system</Th>
                    <Th>
                        Affected components
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th>First discovered</Th>
                </Tr>
            </Thead>
            {images.length === 0 && <EmptyTableResults colSpan={7} />}
            {images.map((image, rowIndex) => {
                const { id, name, operatingSystem, scanTime, imageComponents } = image;
                const vulnerabilities = imageComponents.flatMap(
                    (imageComponent) => imageComponent.imageVulnerabilities
                );
                const topSeverity = getHighestVulnerabilitySeverity(vulnerabilities);
                const isFixableInImage = getIsSomeVulnerabilityFixable(vulnerabilities);
                const { cvss, scoreVersion } = getHighestCvssScore(vulnerabilities);
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
                            <Td dataLabel="CVSS" modifier="nowrap">
                                <CvssFormatted cvss={cvss} scoreVersion={scoreVersion} />
                            </Td>
                            <Td dataLabel="CVE status" modifier="nowrap">
                                <VulnerabilityFixableIconText isFixable={isFixableInImage} />
                            </Td>
                            <Td dataLabel="Operating system">{operatingSystem}</Td>
                            <Td dataLabel="Affected components">
                                {imageComponents.length === 1
                                    ? imageComponents[0].name
                                    : `${imageComponents.length} components`}
                            </Td>
                            <Td dataLabel="First discovered">
                                <DateDistance date={scanTime} />
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
            })}
        </Table>
    );
}

export default AffectedImagesTable;
