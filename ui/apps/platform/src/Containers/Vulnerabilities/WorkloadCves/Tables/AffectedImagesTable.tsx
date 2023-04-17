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
import ImageNameTd from '../components/ImageNameTd';
import { DynamicColumnIcon } from '../components/DynamicIcon';

import ComponentVulnerabilitiesTable, {
    ComponentVulnerability,
    componentVulnerabilitiesFragment,
    imageMetadataContextFragment,
} from './ComponentVulnerabilitiesTable';

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
    topImageVulnerability: {
        severity: string;
        isFixable: boolean;
    } | null;
    imageComponents: ComponentVulnerability[];
};

export const imagesForCveFragment = gql`
    ${imageMetadataContextFragment}
    ${componentVulnerabilitiesFragment}
    fragment ImagesForCVE on Image {
        ...ImageMetadataContext

        operatingSystem
        watchStatus
        scanTime

        topImageVulnerability {
            severity
            isFixable
        }

        imageComponents(query: $query) {
            ...ComponentVulnerabilities
        }
    }
`;

export type AffectedImagesTableProps = {
    className?: string;
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
                    <Th sort={getSortParams('Severity')}>Severity</Th>
                    <Th sort={getSortParams('Fixable')}>
                        Fix status
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th sort={getSortParams('Operating System')}>Operating system</Th>
                    {/* TODO Add sorting for these columns once aggregate sorting is available in BE */}
                    <Th>
                        Affected components
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th>First discovered</Th>
                </Tr>
            </Thead>
            {images.map((image, rowIndex) => {
                const {
                    id,
                    name,
                    operatingSystem,
                    scanTime,
                    topImageVulnerability,
                    imageComponents,
                } = image;
                const topSeverity =
                    topImageVulnerability?.severity ?? 'UNKNOWN_VULNERABILITY_SEVERITY';

                const isFixable = topImageVulnerability?.isFixable ?? false;
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
                                    <ComponentVulnerabilitiesTable
                                        showImage={false}
                                        images={[
                                            {
                                                imageMetadataContext: image,
                                                componentVulnerabilities: image.imageComponents,
                                            },
                                        ]}
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
