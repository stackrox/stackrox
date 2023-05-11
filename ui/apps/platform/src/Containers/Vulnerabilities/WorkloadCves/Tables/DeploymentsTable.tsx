import React from 'react';
import { gql } from '@apollo/client';
import pluralize from 'pluralize';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Button, ButtonVariant } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import { UseURLSortResult } from 'hooks/useURLSort';
import { getEntityPagePath } from '../searchUtils';
import SeverityCountLabels from '../components/SeverityCountLabels';
import { DynamicColumnIcon } from '../components/DynamicIcon';
import EmptyTableResults from '../components/EmptyTableResults';
import DatePhraseTd from '../components/DatePhraseTd';
import TooltipTh from '../components/TooltipTh';

export const deploymentListQuery = gql`
    query getDeploymentList($query: String, $pagination: Pagination) {
        deployments(query: $query, pagination: $pagination) {
            id
            name
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
        critical: { total: number };
        important: { total: number };
        moderate: { total: number };
        low: { total: number };
    };
    clusterName: string;
    namespace: string;
    imageCount: number;
    created: string | null;
};

type DeploymentsTableProps = {
    deployments: Deployment[];
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
};

function DeploymentsTable({ deployments, getSortParams, isFiltered }: DeploymentsTableProps) {
    return (
        <TableComposable borders={false} variant="compact">
            <Thead noWrap>
                {/* TODO: need to double check sorting on columns  */}
                <Tr>
                    <Th sort={getSortParams('Deployment')}>Deployment</Th>
                    <TooltipTh tooltip="CVEs by severity across this deployment">
                        CVEs by severity
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
                    <Th sort={getSortParams('Cluster')}>Cluster</Th>
                    <Th sort={getSortParams('Namespace')}>Namespace</Th>
                    <Th>
                        Images
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th>First discovered</Th>
                </Tr>
            </Thead>
            {deployments.length === 0 && <EmptyTableResults colSpan={6} />}
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
                                        critical={imageCVECountBySeverity.critical.total}
                                        important={imageCVECountBySeverity.important.total}
                                        moderate={imageCVECountBySeverity.moderate.total}
                                        low={imageCVECountBySeverity.low.total}
                                        entity="deployment"
                                    />
                                </Td>
                                <Td>{clusterName}</Td>
                                <Td>{namespace}</Td>
                                <Td>
                                    <Button
                                        variant={ButtonVariant.link}
                                        isInline
                                        component={LinkShim}
                                        href={getEntityPagePath('Deployment', id, {
                                            detailsTab: 'Resources',
                                        })}
                                    >
                                        {imageCount} {pluralize('image', imageCount)}
                                    </Button>
                                </Td>
                                <Td>
                                    <DatePhraseTd date={created} />
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
