import {
    Pagination,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    pluralize,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import type { UseURLPaginationResult } from 'hooks/useURLPagination';
import type { UseURLSortResult } from 'hooks/useURLSort';
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
    'Deployment Risk Priority',
];
export const defaultSortOption = { field: 'Deployment Risk Priority', direction: 'asc' } as const;

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
                    <ToolbarItem>
                        <Title headingLevel="h2">{pluralize(deploymentCount, 'deployment')}</Title>
                    </ToolbarItem>
                    <ToolbarItem align={{ default: 'alignRight' }} variant="pagination">
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
                        <Th width={10} sort={getSortParams('Deployment Risk Priority')}>
                            Priority
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

                            const priorityAsInt = parseInt(deployment.priority, 10);
                            const priorityDisplay =
                                Number.isNaN(priorityAsInt) || priorityAsInt < 1
                                    ? '-'
                                    : priorityAsInt;

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
                                        <Td dataLabel="Priority">{priorityDisplay}</Td>
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
