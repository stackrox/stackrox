import React from 'react';
import { Link } from 'react-router-dom-v5-compat';
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
import {
    generateVisibilityForColumns,
    getHiddenColumnCount,
    ManagedColumns,
} from 'hooks/useManagedColumns';
import SeverityCountLabels from '../../components/SeverityCountLabels';
import { VulnerabilitySeverityLabel } from '../../types';
import useVulnerabilityState from '../hooks/useVulnerabilityState';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';

export const tableId = 'WorkloadCvesDeploymentOverviewTable';

export const defaultColumns = {
    deployment: {
        title: 'Deployment',
        isShownByDefault: true,
        isUntoggleAble: true,
    },
    cvesBySeverity: {
        title: 'CVEs by severity',
        isShownByDefault: true,
    },
    cluster: {
        title: 'Cluster',
        isShownByDefault: true,
    },
    namespace: {
        title: 'Namespace',
        isShownByDefault: true,
    },
    images: {
        title: 'Images',
        isShownByDefault: true,
    },
    firstDiscovered: {
        title: 'First discovered',
        isShownByDefault: true,
    },
} as const;

export const deploymentListQuery = gql`
    query getDeploymentList($query: String, $pagination: Pagination) {
        deployments(query: $query, pagination: $pagination) {
            id
            name
            type
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
                unknown {
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
    type: string;
    imageCVECountBySeverity: {
        critical: { total: number };
        important: { total: number };
        moderate: { total: number };
        low: { total: number };
        unknown: { total: number };
    };
    clusterName: string;
    namespace: string;
    imageCount: number;
    created: string | null;
};

type DeploymentOverviewTableProps = {
    tableState: TableUIState<Deployment>;
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
    filteredSeverities?: VulnerabilitySeverityLabel[];
    onClearFilters: () => void;
    columnVisibilityState: ManagedColumns<keyof typeof defaultColumns>['columns'];
};

function DeploymentOverviewTable({
    tableState,
    getSortParams,
    isFiltered,
    filteredSeverities,
    onClearFilters,
    columnVisibilityState,
}: DeploymentOverviewTableProps) {
    const { urlBuilder } = useWorkloadCveViewContext();
    const vulnerabilityState = useVulnerabilityState();
    const getVisibilityClass = generateVisibilityForColumns(columnVisibilityState);
    const hiddenColumnCount = getHiddenColumnCount(columnVisibilityState);
    const colSpan = Object.values(defaultColumns).length - hiddenColumnCount;

    return (
        <Table borders={false} variant="compact">
            <Thead noWrap>
                <Tr>
                    <Th
                        className={getVisibilityClass('deployment')}
                        sort={getSortParams('Deployment')}
                    >
                        Deployment
                    </Th>
                    <TooltipTh
                        className={getVisibilityClass('cvesBySeverity')}
                        tooltip="CVEs by severity across this deployment"
                    >
                        CVEs by severity
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
                    <Th className={getVisibilityClass('cluster')} sort={getSortParams('Cluster')}>
                        Cluster
                    </Th>
                    <Th
                        className={getVisibilityClass('namespace')}
                        sort={getSortParams('Namespace')}
                    >
                        Namespace
                    </Th>
                    <Th className={getVisibilityClass('images')}>
                        Images
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th
                        className={getVisibilityClass('firstDiscovered')}
                        sort={getSortParams('Created')}
                    >
                        First discovered
                    </Th>
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
                            type,
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
                        const unknownCount = imageCVECountBySeverity.unknown.total;
                        return (
                            <Tbody
                                key={id}
                                style={{
                                    borderBottom: '1px solid var(--pf-v5-c-table--BorderColor)',
                                }}
                            >
                                <Tr>
                                    <Td
                                        className={getVisibilityClass('deployment')}
                                        dataLabel="Deployment"
                                    >
                                        <Link
                                            to={urlBuilder.workloadDetails(
                                                { id, namespace, name, type },
                                                vulnerabilityState
                                            )}
                                        >
                                            <Truncate position="middle" content={name} />
                                        </Link>
                                    </Td>
                                    <Td
                                        dataLabel="CVEs by severity"
                                        className={getVisibilityClass('cvesBySeverity')}
                                    >
                                        <SeverityCountLabels
                                            criticalCount={criticalCount}
                                            importantCount={importantCount}
                                            moderateCount={moderateCount}
                                            lowCount={lowCount}
                                            unknownCount={unknownCount}
                                            entity="deployment"
                                            filteredSeverities={filteredSeverities}
                                        />
                                    </Td>
                                    <Td
                                        dataLabel="Cluster"
                                        className={getVisibilityClass('cluster')}
                                    >
                                        {clusterName}
                                    </Td>
                                    <Td
                                        dataLabel="Namespace"
                                        className={getVisibilityClass('namespace')}
                                    >
                                        {namespace}
                                    </Td>
                                    <Td dataLabel="Images" className={getVisibilityClass('images')}>
                                        <>
                                            {imageCount} {pluralize('image', imageCount)}
                                        </>
                                    </Td>
                                    <Td
                                        dataLabel="First discovered"
                                        className={getVisibilityClass('firstDiscovered')}
                                    >
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

export default DeploymentOverviewTable;
