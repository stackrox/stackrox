import React from 'react';
import { Link } from 'react-router-dom';
import { gql } from '@apollo/client';
import pluralize from 'pluralize';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Truncate } from '@patternfly/react-core';

import { UseURLSortResult } from 'hooks/useURLSort';
import { getEntityPagePath } from '../searchUtils';
import SeverityCountLabels from '../components/SeverityCountLabels';
import { DynamicColumnIcon } from '../components/DynamicIcon';
import EmptyTableResults from '../components/EmptyTableResults';
import DateDistanceTd from '../components/DatePhraseTd';
import TooltipTh from '../components/TooltipTh';
import { VulnerabilitySeverityLabel } from '../types';

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
    filteredSeverities?: VulnerabilitySeverityLabel[];
};

function DeploymentsTable({
    deployments,
    getSortParams,
    isFiltered,
    filteredSeverities,
}: DeploymentsTableProps) {
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
                    <Th sort={getSortParams('Created')}>First discovered</Th>
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
                                <Td>
                                    <Link to={getEntityPagePath('Deployment', id)}>
                                        <Truncate position="middle" content={name} />
                                    </Link>
                                </Td>
                                <Td>
                                    <SeverityCountLabels
                                        criticalCount={criticalCount}
                                        importantCount={importantCount}
                                        moderateCount={moderateCount}
                                        lowCount={lowCount}
                                        entity="deployment"
                                        filteredSeverities={filteredSeverities}
                                    />
                                </Td>
                                <Td>{clusterName}</Td>
                                <Td>{namespace}</Td>
                                <Td>
                                    <>
                                        {imageCount} {pluralize('image', imageCount)}
                                    </>
                                </Td>
                                <Td>
                                    <DateDistanceTd date={created} />
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
