import React from 'react';
import { gql } from '@apollo/client';
import pluralize from 'pluralize';
import { ActionsColumn, TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Flex, Label } from '@patternfly/react-core';

import { UseURLSortResult } from 'hooks/useURLSort';
import { EyeIcon } from '@patternfly/react-icons';
import ImageNameTd from '../components/ImageNameTd';
import SeverityCountLabels from '../components/SeverityCountLabels';
import { DynamicColumnIcon } from '../components/DynamicIcon';
import EmptyTableResults from '../components/EmptyTableResults';
import DateDistanceTd from '../components/DatePhraseTd';
import TooltipTh from '../components/TooltipTh';
import { VulnerabilitySeverityLabel, WatchStatus } from '../types';

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
            operatingSystem
            deploymentCount(query: $query)
            watchStatus
            metadata {
                v1 {
                    created
                }
            }
            scanTime
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
    operatingSystem: string;
    deploymentCount: number;
    watchStatus: WatchStatus;
    metadata: {
        v1: {
            created: string | null;
        } | null;
    } | null;
    scanTime: string | null;
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
    const colSpan = hasWriteAccessForWatchedImage ? 7 : 6;

    return (
        <TableComposable borders={false} variant="compact">
            <Thead noWrap>
                {/* TODO: need to double check sorting on columns  */}
                <Tr>
                    <Th sort={getSortParams('Image')}>Image</Th>
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
                    operatingSystem,
                    deploymentCount,
                    metadata,
                    watchStatus,
                    scanTime,
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
                                borderBottom: '1px solid var(--pf-c-table--BorderColor)',
                            }}
                        >
                            <Tr>
                                <Td dataLabel="Image">
                                    {name ? (
                                        <ImageNameTd name={name} id={id}>
                                            {isWatchedImage && (
                                                <Label
                                                    isCompact
                                                    variant="outline"
                                                    color="grey"
                                                    className="pf-u-mt-xs"
                                                    icon={<EyeIcon />}
                                                >
                                                    Watched image
                                                </Label>
                                            )}
                                        </ImageNameTd>
                                    ) : (
                                        'Image name not available'
                                    )}
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
                                    <DateDistanceTd date={metadata?.v1?.created} asPhrase={false} />
                                </Td>
                                <Td>
                                    <DateDistanceTd date={scanTime} />
                                </Td>
                                {hasWriteAccessForWatchedImage && (
                                    <Td isActionCell>
                                        {name?.tag && (
                                            <ActionsColumn
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
        </TableComposable>
    );
}

export default ImagesTable;
