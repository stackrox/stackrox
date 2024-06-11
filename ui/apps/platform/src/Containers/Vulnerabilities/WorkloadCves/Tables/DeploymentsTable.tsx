import React from 'react';
import { Link } from 'react-router-dom';
import { gql } from '@apollo/client';
import pluralize from 'pluralize';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Truncate } from '@patternfly/react-core';

import { UseURLSortResult } from 'hooks/useURLSort';
import { DynamicColumnIcon } from 'Components/DynamicIcon';
import TooltipTh from 'Components/TooltipTh';
import DateDistance from 'Components/DateDistance';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { TableUIState } from 'utils/getTableUIState';
import { getWorkloadEntityPagePath } from '../../utils/searchUtils';
import SeverityCountLabels from '../../components/SeverityCountLabels';
import { VulnerabilitySeverityLabel } from '../../types';
import useVulnerabilityState from '../hooks/useVulnerabilityState';

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
    tableState: TableUIState<Deployment>;
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
    filteredSeverities?: VulnerabilitySeverityLabel[];
    showCveDetailFields: boolean;
    onClearFilters: () => void;
};

function DeploymentsTable({
    tableState,
    getSortParams,
    isFiltered,
    filteredSeverities,
    showCveDetailFields,
    onClearFilters,
}: DeploymentsTableProps) {
    const vulnerabilityState = useVulnerabilityState();
    const colSpan = showCveDetailFields ? 6 : 5;
    return (
        <Table borders={false} variant="compact">
            <Thead noWrap>
                {/* TODO: need to double check sorting on columns  */}
                <Tr>
                    <Th sort={getSortParams('Deployment')}>Deployment</Th>
                    {showCveDetailFields && (
                        <TooltipTh tooltip="CVEs by severity across this deployment">
                            CVEs by severity
                            {isFiltered && <DynamicColumnIcon />}
                        </TooltipTh>
                    )}
                    <Th sort={getSortParams('Cluster')}>Cluster</Th>
                    <Th sort={getSortParams('Namespace')}>Namespace</Th>
                    <Th>
                        Images
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th sort={getSortParams('Created')}>First discovered</Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={colSpan}
                filteredEmptyProps={{ onClearFilters }}
                emptyProps={{ message: 'No deployments with CVEs were observed in the system' }}
                renderer={({ data }) =>
                    data.map((deployment) => {
                        const {
                            id,
                            name,
                            imageCVECountBySeverity,
                            clusterName,
                            namespace,
                            imageCount,
                            created,
                        } = deployment;
                        const criticalCount = imageCVECountBySeverity.critical.total;
                        const importantCount = imageCVECountBySeverity.important.total;
                        const moderateCount = imageCVECountBySeverity.moderate.total;
                        const lowCount = imageCVECountBySeverity.low.total;
                        return (
                            <Tbody
                                key={id}
                                style={{
                                    borderBottom: '1px solid var(--pf-v5-c-table--BorderColor)',
                                }}
                            >
                                <Tr>
                                    <Td dataLabel="Deployment">
                                        <Link
                                            to={getWorkloadEntityPagePath(
                                                'Deployment',
                                                id,
                                                vulnerabilityState
                                            )}
                                        >
                                            <Truncate position="middle" content={name} />
                                        </Link>
                                    </Td>
                                    {showCveDetailFields && (
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
                                    )}
                                    <Td>{clusterName}</Td>
                                    <Td>{namespace}</Td>
                                    <Td>
                                        <>
                                            {imageCount} {pluralize('image', imageCount)}
                                        </>
                                    </Td>
                                    <Td>
                                        <DateDistance date={created} />
                                    </Td>
                                </Tr>
                            </Tbody>
                        );
                    })
                }
            />
        </Table>
    );
}

export default DeploymentsTable;
