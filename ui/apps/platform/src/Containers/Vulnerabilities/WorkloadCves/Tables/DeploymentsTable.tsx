import React from 'react';
import { gql } from '@apollo/client';
import pluralize from 'pluralize';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Button, ButtonVariant, Tooltip } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import { getDistanceStrictAsPhrase, getDateTime } from 'utils/dateUtils';
import { UseURLSortResult } from 'hooks/useURLSort';
import { getEntityPagePath } from '../searchUtils';
import SeverityCountLabels from '../components/SeverityCountLabels';
import { DynamicColumnIcon } from '../components/DynamicIcon';

export const deploymentListQuery = gql`
    query getDeploymentList($query: String, $pagination: Pagination) {
        deployments(query: $query, pagination: $pagination) {
            id
            name
            imageCVECountBySeverity(query: $query) {
                critical
                important
                moderate
                low
            }
            clusterName
            namespace
            imageCount(query: $query)
            created
        }
    }
`;

export type Deployment = {
    id: string;
    name: string;
    imageCVECountBySeverity: {
        critical: number;
        important: number;
        moderate: number;
        low: number;
    };
    clusterName: string;
    namespace: string;
    imageCount: number;
    created: Date | null;
};

type DeploymentsTableProps = {
    deployments: Deployment[];
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
};

function DeploymentsTable({ deployments, getSortParams, isFiltered }: DeploymentsTableProps) {
    return (
        <TableComposable borders={false} variant="compact">
            <Thead>
                {/* TODO: need to double check sorting on columns  */}
                <Tr>
                    <Th sort={getSortParams('Deployment')}>Deployment</Th>
                    <Th tooltip="CVEs by severity across this deployment">
                        CVEs by severity
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th sort={getSortParams('Cluster')}>Cluster</Th>
                    <Th sort={getSortParams('Namespace')}>Namespace</Th>
                    <Th>
                        Images
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th>First discovered</Th>
                </Tr>
            </Thead>
            {deployments.map(
                ({
                    id,
                    name,
                    imageCVECountBySeverity,
                    clusterName,
                    namespace,
                    imageCount,
                    created,
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
                                    <Button
                                        variant={ButtonVariant.link}
                                        isInline
                                        component={LinkShim}
                                        href={getEntityPagePath('Deployment', id)}
                                    >
                                        {name}
                                    </Button>
                                </Td>
                                <Td>
                                    <SeverityCountLabels
                                        critical={imageCVECountBySeverity.critical}
                                        important={imageCVECountBySeverity.important}
                                        moderate={imageCVECountBySeverity.moderate}
                                        low={imageCVECountBySeverity.low}
                                    />
                                </Td>
                                <Td>{clusterName}</Td>
                                <Td>{namespace}</Td>
                                <Td>
                                    {/* TODO: add modal */}
                                    <Button
                                        variant={ButtonVariant.link}
                                        isInline
                                        component={LinkShim}
                                        href={getEntityPagePath('Deployment', id)}
                                    >
                                        {imageCount} {pluralize('image', imageCount)}
                                    </Button>
                                </Td>
                                <Td>
                                    <Tooltip content={getDateTime(created)}>
                                        <div>{getDistanceStrictAsPhrase(created, new Date())}</div>
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

export default DeploymentsTable;
