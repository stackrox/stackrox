import React from 'react';
import { gql } from '@apollo/client';
import pluralize from 'pluralize';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Button, ButtonVariant, Flex } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import { UseURLSortResult } from 'hooks/useURLSort';
import ImageNameTd from '../components/ImageNameTd';
import { getEntityPagePath } from '../searchUtils';
import SeverityCountLabels from '../components/SeverityCountLabels';
import { DynamicColumnIcon } from '../components/DynamicIcon';
import EmptyTableResults from '../components/EmptyTableResults';
import DatePhraseTd from '../components/DatePhraseTd';
import TooltipTh from '../components/TooltipTh';

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
    watchStatus: 'WATCHED' | 'NOT_WATCHED';
    metadata: {
        v1: {
            created: string | null;
        } | null;
    } | null;
    scanTime: string | null;
};

type ImagesTableProps = {
    images: Image[];
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
};

function ImagesTable({ images, getSortParams, isFiltered }: ImagesTableProps) {
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
                    <Th sort={getSortParams('Operating System')}>Operating system</Th>
                    <Th sort={getSortParams('Deployment Count')}>
                        Deployments
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th sort={getSortParams('Age')}>Age</Th>
                    <Th sort={getSortParams('Scan Time')}>Scan time</Th>
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
                    return (
                        <Tbody
                            key={id}
                            style={{
                                borderBottom: '1px solid var(--pf-c-table--BorderColor)',
                            }}
                        >
                            <Tr>
                                <Td>
                                    {name ? (
                                        <ImageNameTd name={name} id={id} />
                                    ) : (
                                        'Image name not available'
                                    )}
                                </Td>
                                <Td>
                                    <SeverityCountLabels
                                        critical={imageCVECountBySeverity.critical.total}
                                        important={imageCVECountBySeverity.important.total}
                                        moderate={imageCVECountBySeverity.moderate.total}
                                        low={imageCVECountBySeverity.low.total}
                                        entity="image"
                                    />
                                </Td>
                                <Td>{operatingSystem}</Td>
                                <Td>
                                    {deploymentCount > 0 ? (
                                        <Button
                                            variant={ButtonVariant.link}
                                            isInline
                                            component={LinkShim}
                                            href={getEntityPagePath('Image', id, {
                                                detailsTab: 'Resources',
                                            })}
                                        >
                                            {deploymentCount}{' '}
                                            {pluralize('deployment', deploymentCount)}
                                        </Button>
                                    ) : (
                                        <Flex>
                                            <div>0 deployments</div>
                                            {/* TODO: double check on what this links to */}
                                            <span>({`${watchStatus}`} image)</span>
                                        </Flex>
                                    )}
                                </Td>
                                <Td>
                                    <DatePhraseTd date={metadata?.v1?.created} />
                                </Td>
                                <Td>
                                    <DatePhraseTd date={scanTime} />
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
