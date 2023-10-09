import React from 'react';
import { gql } from '@apollo/client';
import {
    ExpandableRowContent,
    TableComposable,
    Tbody,
    Td,
    Thead,
    Th,
    Tr,
} from '@patternfly/react-table';

import useSet from 'hooks/useSet';
import { UseURLSortResult } from 'hooks/useURLSort';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import {
    getAnyVulnerabilityIsFixable,
    getHighestCvssScore,
    getHighestVulnerabilitySeverity,
} from './table.utils';
import ImageNameTd from '../components/ImageNameTd';
import { DynamicColumnIcon } from '../components/DynamicIcon';

import ImageComponentVulnerabilitiesTable, {
    ImageComponentVulnerability,
    imageComponentVulnerabilitiesFragment,
    imageMetadataContextFragment,
} from './ImageComponentVulnerabilitiesTable';
import EmptyTableResults from '../components/EmptyTableResults';
import DateDistanceTd from '../components/DatePhraseTd';
import CvssTd from '../components/CvssTd';
import { WatchStatus } from '../types';

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
};

function AffectedImagesTable({ images, getSortParams, isFiltered }: AffectedImagesTableProps) {
    const expandedRowSet = useSet<string>();

    return (
        <TableComposable variant="compact">
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
                const topSeverity = getHighestVulnerabilitySeverity(imageComponents);
                const isFixable = getAnyVulnerabilityIsFixable(imageComponents);
                const { cvss, scoreVersion } = getHighestCvssScore(imageComponents);

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
                                    <ImageNameTd name={name} id={id} />
                                ) : (
                                    'Image name not available'
                                )}
                            </Td>
                            <Td dataLabel="CVE severity" modifier="nowrap">
                                <VulnerabilitySeverityIconText severity={topSeverity} />
                            </Td>
                            <Td dataLabel="CVSS" modifier="nowrap">
                                <CvssTd cvss={cvss} scoreVersion={scoreVersion} />
                            </Td>
                            <Td dataLabel="CVE status" modifier="nowrap">
                                <VulnerabilityFixableIconText isFixable={isFixable} />
                            </Td>
                            <Td dataLabel="Operating system">{operatingSystem}</Td>
                            <Td dataLabel="Affected components">
                                {imageComponents.length === 1
                                    ? imageComponents[0].name
                                    : `${imageComponents.length} components`}
                            </Td>
                            <Td dataLabel="First discovered">
                                <DateDistanceTd date={scanTime} />
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
        </TableComposable>
    );
}

export default AffectedImagesTable;
