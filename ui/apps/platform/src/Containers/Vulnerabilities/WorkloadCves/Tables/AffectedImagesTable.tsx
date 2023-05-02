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

import { FixableIcon, NotFixableIcon } from 'Components/PatternFly/FixabilityIcons';
import SeverityIcons from 'Components/PatternFly/SeverityIcons';
import useSet from 'hooks/useSet';
import { UseURLSortResult } from 'hooks/useURLSort';
import { vulnerabilitySeverityLabels } from 'messages/common';
import { getDistanceStrictAsPhrase } from 'utils/dateUtils';
import { severityRankings } from 'constants/vulnerabilities';
import { VulnerabilitySeverity, isVulnerabilitySeverity } from 'types/cve.proto';
import ImageNameTd from '../components/ImageNameTd';
import { DynamicColumnIcon } from '../components/DynamicIcon';

import ImageComponentVulnerabilitiesTable, {
    ImageComponentVulnerability,
    imageComponentVulnerabilitiesFragment,
    imageMetadataContextFragment,
} from './ImageComponentVulnerabilitiesTable';
import EmptyTableResults from '../components/EmptyTableResults';

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
    watchStatus: 'WATCHED' | 'NOT_WATCHED';
    scanTime: Date | null;
    imageComponents: ImageComponentVulnerability[];
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
            ...ImageComponentVulnerabilities
        }
    }
`;

/**
 * Get the highest severity of any vulnerability in the image.
 */
function getVulnerabilitySeverity(
    imageComponents: ImageComponentVulnerability[]
): VulnerabilitySeverity {
    let topSeverity: VulnerabilitySeverity = 'UNKNOWN_VULNERABILITY_SEVERITY';
    imageComponents.forEach((component) => {
        component.imageVulnerabilities.forEach(({ severity }) => {
            if (
                isVulnerabilitySeverity(severity) &&
                severityRankings[severity] > severityRankings[topSeverity]
            ) {
                topSeverity = severity;
            }
        });
    });
    return topSeverity;
}

/**
 * Get whether or not the image has any fixable vulnerabilities.
 */
function getIsFixable(imageComponents: ImageComponentVulnerability[]) {
    return imageComponents.find((component) =>
        component.imageVulnerabilities.find(({ fixedByVersion }) => fixedByVersion !== '')
    );
}

export type AffectedImagesTableProps = {
    images: ImageForCve[];
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
};

function AffectedImagesTable({ images, getSortParams, isFiltered }: AffectedImagesTableProps) {
    const expandedRowSet = useSet<string>();

    return (
        // TODO UX question - Collapse to cards, or allow headers to overflow?
        // <TableComposable gridBreakPoint="grid-xl">
        <TableComposable variant="compact">
            <Thead>
                <Tr>
                    <Th>{/* Header for expanded column */}</Th>
                    <Th sort={getSortParams('Image')}>Image</Th>
                    <Th>Severity</Th>
                    <Th>
                        Fix status
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
                const topSeverity = getVulnerabilitySeverity(imageComponents);
                const isFixable = getIsFixable(imageComponents);
                const FixabilityIcon = isFixable ? FixableIcon : NotFixableIcon;

                const SeverityIcon = SeverityIcons[topSeverity];
                const severityLabel = vulnerabilitySeverityLabels[topSeverity];
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
                            <Td dataLabel="Severity">
                                <span>
                                    {SeverityIcon && (
                                        <SeverityIcon className="pf-u-display-inline" />
                                    )}
                                    {severityLabel && (
                                        <span className="pf-u-pl-sm">{severityLabel}</span>
                                    )}
                                </span>
                            </Td>
                            <Td dataLabel="Fix status">
                                <span>
                                    <FixabilityIcon className="pf-u-display-inline" />
                                    <span className="pf-u-pl-sm">
                                        {isFixable ? 'Fixable' : 'Not fixable'}
                                    </span>
                                </span>
                            </Td>
                            <Td dataLabel="Operating system">{operatingSystem}</Td>
                            <Td dataLabel="Affected components">
                                {imageComponents.length === 1
                                    ? imageComponents[0].name
                                    : `${imageComponents.length} components`}
                            </Td>
                            <Td dataLabel="First discovered">
                                {getDistanceStrictAsPhrase(scanTime, new Date())}
                            </Td>
                        </Tr>
                        <Tr isExpanded={isExpanded}>
                            <Td />
                            <Td colSpan={6}>
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
