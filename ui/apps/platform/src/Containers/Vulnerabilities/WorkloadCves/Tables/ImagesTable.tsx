import React from 'react';
import pluralize from 'pluralize';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Button, ButtonVariant, Flex, Tooltip } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import { getDistanceStrictAsPhrase, getDateTime } from 'utils/dateUtils';
import { UseURLSortResult } from 'hooks/useURLSort';
import { getEntityPagePath } from '../searchUtils';
import SeverityCountLabels from '../SeverityCountLabels';

type ImagesTableProps = {
    images: {
        id: string;
        name: {
            registry: string;
            remote: string;
            tag: string;
        };
        imageCVECountBySeverity: {
            critical: number;
            important: number;
            moderate: number;
            low: number;
        };
        operatingSystem: string;
        deploymentCount: number;
        watchStatus: string;
        metadata?: {
            v1?: {
                created?: Date;
            };
        };
        scanTime?: Date;
    }[];
    getSortParams: UseURLSortResult['getSortParams'];
};

function ImagesTable({ images, getSortParams }: ImagesTableProps) {
    return (
        <TableComposable borders={false} variant="compact">
            <Thead>
                {/* TODO: need to add sorting to columns  */}
                <Tr>
                    <Th sort={getSortParams('Images')}>Image</Th>
                    <Th sort={getSortParams('CVE')}>CVEs by severity</Th>
                    <Th sort={getSortParams('Operating system')}>Operating system</Th>
                    <Th sort={getSortParams('Deployment count')}>Deployments</Th>
                    <Th sort={getSortParams('Age')}>Age</Th>
                    <Th sort={getSortParams('Scan time')}>Scan time</Th>
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
                                {/* TODO: need to add path */}
                                <Td>
                                    <Flex
                                        direction={{ default: 'column' }}
                                        spaceItems={{ default: 'spaceItemsXs' }}
                                    >
                                        <Button
                                            variant={ButtonVariant.link}
                                            isInline
                                            component={LinkShim}
                                            href={getEntityPagePath('Image', id)}
                                        >
                                            {name.remote}
                                        </Button>
                                        <span>in {`"${name.registry || 'unknown'}"`}</span>
                                    </Flex>
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
