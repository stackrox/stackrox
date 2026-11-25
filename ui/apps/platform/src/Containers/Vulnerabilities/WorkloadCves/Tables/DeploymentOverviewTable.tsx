import { Link } from 'react-router-dom-v5-compat';
import pluralize from 'pluralize';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Truncate } from '@patternfly/react-core';

import type { UseURLSortResult } from 'hooks/useURLSort';
import { DynamicColumnIcon } from 'Components/DynamicIcon';
import TooltipTh from 'Components/TooltipTh';
import DateDistance from 'Components/DateDistance';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import type { TableUIState } from 'utils/getTableUIState';
import { generateVisibilityForColumns, getHiddenColumnCount } from 'hooks/useManagedColumns';
import type { ManagedColumns } from 'hooks/useManagedColumns';
import type { VulnMgmtDeploymentListItem } from 'services/VulnMgmtService';
import useVulnerabilityState from '../hooks/useVulnerabilityState';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';

export const tableId = 'WorkloadCvesDeploymentOverviewTable';

export const defaultColumns = {
    deployment: {
        title: 'Deployment',
        isShownByDefault: true,
        isUntoggleAble: true,
    },
    matchingCveCount: {
        title: 'Matching CVEs',
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

export type DeploymentOverviewTableProps = {
    tableState: TableUIState<VulnMgmtDeploymentListItem>;
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
    onClearFilters: () => void;
    columnVisibilityState: ManagedColumns<keyof typeof defaultColumns>['columns'];
};

function DeploymentOverviewTable({
    tableState,
    getSortParams,
    isFiltered,
    onClearFilters,
    columnVisibilityState,
}: DeploymentOverviewTableProps) {
    const { urlBuilder } = useWorkloadCveViewContext();
    const vulnerabilityState = useVulnerabilityState();
    const getVisibilityClass = generateVisibilityForColumns(columnVisibilityState);
    const hiddenColumnCount = getHiddenColumnCount(columnVisibilityState);
    const colSpan = Object.values(defaultColumns).length - hiddenColumnCount;

    return (
        <Table variant="compact">
            <Thead noWrap>
                <Tr>
                    <Th
                        className={getVisibilityClass('deployment')}
                        sort={getSortParams('Deployment')}
                    >
                        Deployment
                    </Th>
                    <TooltipTh
                        className={getVisibilityClass('matchingCveCount')}
                        tooltip="CVEs in this deployment matching the active severity and filters"
                    >
                        Matching CVEs
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
                            cluster,
                            namespace,
                            createdAt,
                            imageCount,
                            matchedCveCount,
                        } = deployment;
                        return (
                            <Tbody key={id}>
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
                                        dataLabel="Matching CVEs"
                                        className={getVisibilityClass('matchingCveCount')}
                                    >
                                        {matchedCveCount}
                                    </Td>
                                    <Td
                                        dataLabel="Cluster"
                                        className={getVisibilityClass('cluster')}
                                    >
                                        {cluster}
                                    </Td>
                                    <Td
                                        dataLabel="Namespace"
                                        className={getVisibilityClass('namespace')}
                                    >
                                        {namespace}
                                    </Td>
                                    <Td
                                        dataLabel="Images"
                                        className={getVisibilityClass('images')}
                                        modifier="nowrap"
                                    >
                                        <>
                                            {imageCount} {pluralize('image', imageCount)}
                                        </>
                                    </Td>
                                    <Td
                                        dataLabel="First discovered"
                                        className={getVisibilityClass('firstDiscovered')}
                                        modifier="nowrap"
                                    >
                                        <DateDistance date={createdAt} />
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
