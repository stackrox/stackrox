import { Label, Pagination, Toolbar, ToolbarContent, ToolbarItem } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import type { UseURLPaginationResult } from 'hooks/useURLPagination';
import type { UseURLSortResult } from 'hooks/useURLSort';
import type { RiskPriorityClassification } from 'types/risk.proto';
import type { ApiSortOption, SearchFilter } from 'types/search';
import { getDateTime } from 'utils/dateUtils';
import { getTableUIState } from 'utils/getTableUIState';

import { DeploymentNameColumn } from './DeploymentNameColumn';
import useDeploymentsCount from './useDeploymentsCount';
import useDeploymentsWithProcessInfo from './useDeploymentsWithProcessInfo';

export const sortFields = [
    'Deployment',
    'Created',
    'Cluster',
    'Namespace',
    'Risk Priority Classification',
    'Deployment Risk Priority',
];
export const defaultSortOption = {
    field: 'Risk Priority Classification',
    direction: 'asc',
} as const;

const riskClassLabels: Record<RiskPriorityClassification, string> = {
    RISK_PRIORITY_UNKNOWN: 'Unknown',
    RISK_PRIORITY_RESPOND_NOW: 'Respond now',
    RISK_PRIORITY_INVESTIGATE_NOW: 'Investigate now',
    RISK_PRIORITY_REMEDIATE_NOW: 'Remediate now',
    RISK_PRIORITY_REMEDIATE: 'Remediate',
    RISK_PRIORITY_IMPROVE_POSTURE: 'Improve posture',
};

const riskClassColors: Record<
    RiskPriorityClassification,
    'red' | 'orange' | 'yellow' | 'blue' | 'green' | 'grey'
> = {
    RISK_PRIORITY_UNKNOWN: 'grey',
    RISK_PRIORITY_RESPOND_NOW: 'red',
    RISK_PRIORITY_INVESTIGATE_NOW: 'orange',
    RISK_PRIORITY_REMEDIATE_NOW: 'red',
    RISK_PRIORITY_REMEDIATE: 'yellow',
    RISK_PRIORITY_IMPROVE_POSTURE: 'blue',
};

type RiskTablePanelProps = {
    sortOption: ApiSortOption;
    getSortParams: UseURLSortResult['getSortParams'];
    searchFilter: SearchFilter;
    onSearchFilterChange: (newSearchFilter: SearchFilter) => void;
    pagination: UseURLPaginationResult;
};

function RiskTablePanel({
    sortOption,
    getSortParams,
    searchFilter,
    onSearchFilterChange,
    pagination,
}: RiskTablePanelProps) {
    const { page, perPage, setPage, setPerPage } = pagination;

    const { data, error, isLoading } = useDeploymentsWithProcessInfo({
        searchFilter,
        sortOption,
        page,
        perPage,
    });

    const { data: deploymentCount = 0 } = useDeploymentsCount({
        searchFilter,
    });

    const tableState = getTableUIState({ isLoading, data, error, searchFilter });

    return (
        <div>
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem align={{ default: 'alignEnd' }} variant="pagination">
                        <Pagination
                            itemCount={deploymentCount}
                            page={page}
                            onSetPage={(_, newPage) => setPage(newPage)}
                            perPage={perPage}
                            onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            <Table variant="compact">
                <Thead noWrap>
                    <Tr>
                        <Th width={25} sort={getSortParams('Deployment')}>
                            Name
                        </Th>
                        <Th width={25} sort={getSortParams('Created')}>
                            Created
                        </Th>
                        <Th sort={getSortParams('Cluster')}>Cluster</Th>
                        <Th sort={getSortParams('Namespace')}>Namespace</Th>
                        <Th width={10} sort={getSortParams('Risk Priority Classification')}>
                            Risk class
                        </Th>
                    </Tr>
                </Thead>
                <TbodyUnified
                    tableState={tableState}
                    colSpan={5}
                    emptyProps={{ message: 'No results found' }}
                    filteredEmptyProps={{ onClearFilters: () => onSearchFilterChange({}) }}
                    renderer={({ data }) =>
                        data.map((deploymentWithProcessInfo) => {
                            const { deployment } = deploymentWithProcessInfo;

                            const riskClass =
                                deployment.riskPriorityClassification ?? 'RISK_PRIORITY_UNKNOWN';

                            return (
                                <Tbody key={deployment.id}>
                                    <Tr>
                                        <Td dataLabel="Name">
                                            <DeploymentNameColumn
                                                original={deploymentWithProcessInfo}
                                            />
                                        </Td>
                                        <Td dataLabel="Created">
                                            {getDateTime(deployment.created)}
                                        </Td>
                                        <Td dataLabel="Cluster">{deployment.cluster}</Td>
                                        <Td dataLabel="Namespace">{deployment.namespace}</Td>
                                        <Td dataLabel="Risk class">
                                            <Label color={riskClassColors[riskClass]}>
                                                {riskClassLabels[riskClass]}
                                            </Label>
                                        </Td>
                                    </Tr>
                                </Tbody>
                            );
                        })
                    }
                />
            </Table>
        </div>
    );
}

export default RiskTablePanel;
