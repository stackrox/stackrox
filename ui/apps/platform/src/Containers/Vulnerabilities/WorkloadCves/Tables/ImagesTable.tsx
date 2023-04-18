import React from 'react';
import { gql } from '@apollo/client';
import pluralize from 'pluralize';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Button, ButtonVariant, Flex, Tooltip } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import { getDistanceStrictAsPhrase, getDateTime } from 'utils/dateUtils';
import { UseURLSortResult } from 'hooks/useURLSort';
import ImageNameTd from '../components/ImageNameTd';
import { getEntityPagePath } from '../searchUtils';
import SeverityCountLabels from '../components/SeverityCountLabels';
import { DynamicColumnIcon } from '../components/DynamicIcon';

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
                critical
                important
                moderate
                low
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
        critical: number;
        important: number;
        moderate: number;
        low: number;
    };
    operatingSystem: string;
    deploymentCount: number;
    watchStatus: 'WATCHED' | 'NOT_WATCHED';
    metadata: {
        v1: {
            created: Date | null;
        } | null;
    } | null;
    scanTime: Date | null;
};

type ImagesTableProps = {
    images: Image[];
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
};

function ImagesTable({ images, getSortParams, isFiltered }: ImagesTableProps) {
    return (
        <TableComposable borders={false} variant="compact">
            <Thead>
                {/* TODO: need to double check sorting on columns  */}
                <Tr>
                    <Th sort={getSortParams('Image')}>Image</Th>
                    <Th tooltip="CVEs by severity across this image">
                        CVEs by severity
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th sort={getSortParams('Operating System')}>Operating system</Th>
                    <Th sort={getSortParams('Deployment Count')}>
                        Deployments
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th sort={getSortParams('Age')}>Age</Th>
                    <Th sort={getSortParams('Scan Time')}>Scan time</Th>
                </Tr>
            </Thead>
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
                                        critical={imageCVECountBySeverity.critical}
                                        important={imageCVECountBySeverity.important}
                                        moderate={imageCVECountBySeverity.moderate}
                                        low={imageCVECountBySeverity.low}
                                    />
                                </Td>
                                <Td>{operatingSystem}</Td>
                                <Td>
                                    {/* TODO: add modal */}
                                    {deploymentCount > 0 ? (
                                        <Button
                                            variant={ButtonVariant.link}
                                            isInline
                                            component={LinkShim}
                                            href={getEntityPagePath('Deployment', id)}
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
                                    <Tooltip content={getDateTime(metadata?.v1?.created)}>
                                        <div>
                                            {getDistanceStrictAsPhrase(
                                                metadata?.v1?.created,
                                                new Date()
                                            )}
                                        </div>
                                    </Tooltip>
                                </Td>
                                <Td>
                                    <Tooltip content={getDateTime(scanTime)}>
                                        <div>{getDistanceStrictAsPhrase(scanTime, new Date())}</div>
                                    </Tooltip>
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
