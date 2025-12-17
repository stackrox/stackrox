import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import TableHeader from 'Components/TableHeader';
import { PanelBody, PanelHead, PanelHeadEnd, PanelNew } from 'Components/Panel';
import TablePagination from 'Components/TablePagination';
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
    isViewFiltered: boolean;
    sortOption: ApiSortOption;
    getSortParams: UseURLSortResult['getSortParams'];
    searchFilter: SearchFilter;
    onSearchFilterChange: (newSearchFilter: SearchFilter) => void;
    pagination: UseURLPaginationResult;
};

function RiskTablePanel({
    isViewFiltered,
    sortOption,
    getSortParams,
    searchFilter,
    onSearchFilterChange,
    pagination,
}: RiskTablePanelProps) {
    const { page, perPage, setPage } = pagination;

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
        <PanelNew testid="panel">
            <PanelHead>
                <TableHeader
                    length={deploymentCount}
                    type="deployment"
                    isViewFiltered={isViewFiltered}
                />
                <PanelHeadEnd>
                    <TablePagination
                        page={page - 1}
                        dataLength={deploymentCount}
                        pageSize={perPage}
                        setPage={(newPage) => setPage(newPage + 1)}
                    />
                </PanelHeadEnd>
            </PanelHead>
            <PanelBody>
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
            </PanelBody>
        </PanelNew>
    );
}

export default RiskTablePanel;
