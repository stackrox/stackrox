import React from 'react';
import { gql } from '@apollo/client';
import pluralize from 'pluralize';
import { ActionsColumn, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Flex, Label } from '@patternfly/react-core';

import { UseURLSortResult } from 'hooks/useURLSort';
import { EyeIcon } from '@patternfly/react-icons';
import { DynamicColumnIcon } from 'Components/DynamicIcon';
import TooltipTh from 'Components/TooltipTh';
import DateDistance from 'Components/DateDistance';
import ImageNameLink from '../components/ImageNameLink';
import SeverityCountLabels from '../../components/SeverityCountLabels';
import EmptyTableResults from '../components/EmptyTableResults';
import { VulnerabilitySeverityLabel, WatchStatus } from '../../types';
import ImageScanningErrorLabel from '../components/ImageScanningErrorLabelLayout';

export const imageListQuery = gql`
    query getImageList($query: String, $pagination: Pagination) {
        images(query: $query, pagination: $pagination) {
            id
            name {
                registry
                remote
                tag
            }
            imageCVECountBySeverity(query: $query) {
                critical {
                    total
                }
                important {
                    total
                }
                moderate {
                    total
                }
                low {
                    total
                }
            }
            priority
            operatingSystem
            deploymentCount(query: $query)
            watchStatus
            metadata {
                v1 {
                    created
                }
            }
            scanTime
            scanNotes
            notes
        }
    }
`;

type Image = {
    id: string;
    name: {
        registry: string;
        remote: string;
        tag: string;
    } | null;
    imageCVECountBySeverity: {
        critical: { total: number };
        important: { total: number };
        moderate: { total: number };
        low: { total: number };
    };
    priority: number;
    operatingSystem: string;
    deploymentCount: number;
    watchStatus: WatchStatus;
    metadata: {
        v1: {
            created: string | null;
        } | null;
    } | null;
    scanTime: string | null;
    scanNotes: string[];
    notes: string[];
};

export type ImagesTableProps = {
    images: Image[];
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
    filteredSeverities?: VulnerabilitySeverityLabel[];
    hasWriteAccessForWatchedImage: boolean;
    onWatchImage: (imageName: string) => void;
    onUnwatchImage: (imageName: string) => void;
};

function ImagesTable({
    images,
    getSortParams,
    isFiltered,
    filteredSeverities,
    hasWriteAccessForWatchedImage,
    onWatchImage,
    onUnwatchImage,
}: ImagesTableProps) {
    const colSpan = hasWriteAccessForWatchedImage ? 8 : 7;

    return (
        <Table borders={false} variant="compact">
            <Thead noWrap>
                {/* TODO: need to double check sorting on columns  */}
                <Tr>
                    <Th sort={getSortParams('Image')}>Image</Th>
                    <Th sort={getSortParams('Image Risk Priority')}>Risk priority</Th>
                    <TooltipTh tooltip="CVEs by severity across this image">
                        CVEs by severity
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
                    <Th sort={getSortParams('Image OS')}>Operating system</Th>
                    <Th>
                        Deployments
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th sort={getSortParams('Image created time')}>Age</Th>
                    <Th sort={getSortParams('Image scan time')}>Scan time</Th>
                    {hasWriteAccessForWatchedImage && <Th aria-label="Image action menu" />}
                </Tr>
            </Thead>
            {images.length === 0 && <EmptyTableResults colSpan={colSpan} />}
            {images.map(
                ({
                    id,
                    name,
                    imageCVECountBySeverity,
                    priority,
                    operatingSystem,
                    deploymentCount,
                    metadata,
                    watchStatus,
                    scanTime,
                    scanNotes,
                    notes,
                }) => {
                    const criticalCount = imageCVECountBySeverity.critical.total;
                    const importantCount = imageCVECountBySeverity.important.total;
                    const moderateCount = imageCVECountBySeverity.moderate.total;
                    const lowCount = imageCVECountBySeverity.low.total;

                    const isWatchedImage = watchStatus === 'WATCHED';
                    const watchImageMenuText = isWatchedImage ? 'Unwatch image' : 'Watch image';
                    const watchImageMenuAction = isWatchedImage ? onUnwatchImage : onWatchImage;

                    return (
                        <Tbody
                            key={id}
                            style={{
                                borderBottom: '1px solid var(--pf-v5-c-table--BorderColor)',
                            }}
                        >
                            <Tr>
                                <Td dataLabel="Image">
                                    {name ? (
                                        <ImageNameLink name={name} id={id}>
                                            {isWatchedImage && (
                                                <Label
                                                    isCompact
                                                    variant="outline"
                                                    color="grey"
                                                    className="pf-v5-u-mt-xs"
                                                    icon={<EyeIcon />}
                                                >
                                                    Watched image
                                                </Label>
                                            )}
                                            {(notes.length !== 0 || scanNotes.length !== 0) && (
                                                <ImageScanningErrorLabel
                                                    imageNotes={notes}
                                                    scanNotes={scanNotes}
                                                />
                                            )}
                                        </ImageNameLink>
                                    ) : (
                                        'Image name not available'
                                    )}
                                </Td>
                                <Td
                                    dataLabel="Risk priority"
                                    className="pf-v5-u-pr-2xl pf-v5-u-text-align-center-on-md"
                                >
                                    {priority}
                                </Td>
                                <Td dataLabel="CVEs by severity">
                                    <SeverityCountLabels
                                        criticalCount={criticalCount}
                                        importantCount={importantCount}
                                        moderateCount={moderateCount}
                                        lowCount={lowCount}
                                        entity="image"
                                        filteredSeverities={filteredSeverities}
                                    />
                                </Td>
                                <Td>{operatingSystem}</Td>
                                <Td modifier="nowrap">
                                    {deploymentCount > 0 ? (
                                        <>
                                            {deploymentCount}{' '}
                                            {pluralize('deployment', deploymentCount)}
                                        </>
                                    ) : (
                                        <Flex>
                                            <div>0 deployments</div>
                                        </Flex>
                                    )}
                                </Td>
                                <Td>
                                    <DateDistance date={metadata?.v1?.created} asPhrase={false} />
                                </Td>
                                <Td>
                                    <DateDistance date={scanTime} />
                                </Td>
                                {hasWriteAccessForWatchedImage && (
                                    <Td isActionCell>
                                        {name?.tag && (
                                            <ActionsColumn
                                                // menuAppendTo={() => document.body}
                                                items={[
                                                    {
                                                        title: watchImageMenuText,
                                                        onClick: () =>
                                                            watchImageMenuAction(
                                                                `${name.registry}/${name.remote}:${name.tag}`
                                                            ),
                                                    },
                                                ]}
                                            />
                                        )}
                                    </Td>
                                )}
                            </Tr>
                        </Tbody>
                    );
                }
            )}
        </Table>
    );
}

export default ImagesTable;
