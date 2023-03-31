import React from 'react';
import { gql } from '@apollo/client';
import { Button, ButtonVariant, Flex } from '@patternfly/react-core';
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
import LinkShim from 'Components/PatternFly/LinkShim';
import SeverityIcons from 'Components/PatternFly/SeverityIcons';
import useSet from 'hooks/useSet';
import { UseURLSortResult } from 'hooks/useURLSort';
import { vulnerabilitySeverityLabels } from 'messages/common';
import { getDistanceStrictAsPhrase } from 'utils/dateUtils';

import { DynamicColumnIcon } from '../DynamicIcon';
import { getEntityPagePath } from '../searchUtils';

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
    imageComponentCount: number;
    imageComponents: {
        name: string;
        version: string;
        location: string;
        layerIndex: number | null;
        imageVulnerabilities: {
            severity: string;
            fixedByVersion: string;
        }[];
    }[];
};

export const imagesForCveFragment = gql`
    fragment ImagesForCVE on Image {
        id
        name {
            registry
            remote
            tag
        }
        metadata {
            v1 {
                layers {
                    instruction
                    value
                }
                created
            }
        }

        operatingSystem
        watchStatus
        scanTime

        topImageVulnerability {
            severity
            isFixable
        }

        imageComponentCount(query: $query)
        imageComponents(query: $query, pagination: $imageComponentPagination) {
            name
            version
            location
            layerIndex
            imageVulnerabilities(query: $query) {
                severity # same for all components in an image
                fixedByVersion
            }
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
        <TableComposable>
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
            {images.map(
                (
                    { id, name, operatingSystem, scanTime, topImageVulnerability, imageComponents },
                    rowIndex
                ) => {
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
                                        <Flex
                                            direction={{ default: 'column' }}
                                            spaceItems={{ default: 'spaceItemsNone' }}
                                        >
                                            <Button
                                                variant={ButtonVariant.link}
                                                isInline
                                                component={LinkShim}
                                                href={getEntityPagePath('Image', id)}
                                            >
                                                {name.remote}
                                            </Button>{' '}
                                            <span className="pf-u-color-400 pf-u-font-size-sm">
                                                in {name.registry}
                                            </span>
                                        </Flex>
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
                                    {/* TODO Is this the correct field? It differs from the field on the CVE page. */}
                                    {getDistanceStrictAsPhrase(scanTime, new Date())}
                                </Td>
                            </Tr>
                            <Tr isExpanded={isExpanded}>
                                <Td />
                                <Td colSpan={5}>
                                    <ExpandableRowContent>TODO</ExpandableRowContent>
                                </Td>
                            </Tr>
                        </Tbody>
                    );
                }
            )}
        </TableComposable>
    );
}

export default AffectedImagesTable;
