import React from 'react';
import pluralize from 'pluralize';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Flex } from '@patternfly/react-core';

import { UseURLSortResult } from 'hooks/useURLSort';
import { graphql } from 'generated/graphql-codegen';
import { GetImageListQuery } from 'generated/graphql-codegen/graphql';
import ImageNameTd from '../components/ImageNameTd';
import SeverityCountLabels from '../components/SeverityCountLabels';
import { DynamicColumnIcon } from '../components/DynamicIcon';
import EmptyTableResults from '../components/EmptyTableResults';
import DateDistanceTd from '../components/DatePhraseTd';
import TooltipTh from '../components/TooltipTh';
import { VulnerabilitySeverityLabel, watchStatusLabel } from '../types';

export const imageListQuery = graphql(/* GraphQL */ `
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
`);

type ImagesTableProps = {
    images: GetImageListQuery['images'];
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
    filteredSeverities?: VulnerabilitySeverityLabel[];
};

function ImagesTable({ images, getSortParams, isFiltered, filteredSeverities }: ImagesTableProps) {
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
                </Tr>
            </Thead>
            {images.length === 0 && <EmptyTableResults colSpan={6} />}
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
                                        <ImageNameTd name={name} id={id} />
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
                                <Td>
                                    {deploymentCount > 0 ? (
                                        <>
                                            {deploymentCount}{' '}
                                            {pluralize('deployment', deploymentCount)}
                                        </>
                                    ) : (
                                        <Flex>
                                            <div>0 deployments</div>
                                            {/* TODO: double check on what this links to */}
                                            <span>
                                                ({`${watchStatusLabel[watchStatus]}`} image)
                                            </span>
                                        </Flex>
                                    )}
                                </Td>
                                <Td>
                                    <DateDistanceTd date={metadata?.v1?.created} asPhrase={false} />
                                </Td>
                                <Td>
                                    <DateDistanceTd date={scanTime} />
                                </Td>
                            </Tr>
                        </Tbody>
                    );
                }
            )}
        </TableComposable>
    );
}

export default ImagesTable;
