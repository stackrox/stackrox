import NoResultsMessage from 'Components/NoResultsMessage';
import Table from 'Components/TableV2';

import type { ListDeploymentWithProcessInfo } from 'services/DeploymentsService';
import type { UseURLSortResult } from 'hooks/useURLSort';
import type { SortOption } from 'types/table';
import { sortDate, sortValue } from 'sorters/sorters';
import { getDateTime } from 'utils/dateUtils';

import { DeploymentNameColumn } from './DeploymentNameColumn';

const riskTableColumnDescriptors = [
    {
        Header: 'Name',
        accessor: 'deployment.name',
        searchField: 'Deployment',
        Cell: DeploymentNameColumn,
    },
    {
        Header: 'Created',
        accessor: 'deployment.created',
        searchField: 'Created',
        Cell: ({ value }) => <span>{getDateTime(value)}</span>,
        sortMethod: sortDate,
    },
    {
        Header: 'Cluster',
        searchField: 'Cluster',
        accessor: 'deployment.cluster',
    },
    {
        Header: 'Namespace',
        searchField: 'Namespace',
        accessor: 'deployment.namespace',
    },
    {
        Header: 'Priority',
        searchField: 'Deployment Risk Priority',
        accessor: 'deployment.priority',
        Cell: ({ value }: { value: string }) => {
            const asInt = parseInt(value, 10);
            return Number.isNaN(asInt) || asInt < 1 ? '-' : value;
        },
        sortMethod: sortValue,
    },
];

function convertTableSortToURLSetterSort(state): SortOption | null {
    let sortOption: SortOption | null = null;
    if (state.sorted.length && state.sorted[0].id) {
        const column = riskTableColumnDescriptors.find(
            (col) => col.accessor === state.sorted[0].id
        );
        sortOption = {
            // TODO we should be able to assert that column.searchField is not undefined after migrating away
            // from the legacy TableV2 and descriptor pattern
            field: column?.searchField ?? '',
            direction: state.sorted[0].desc ? 'desc' : 'asc',
        };
    }
    return sortOption;
}

type RiskTableProps = {
    currentDeployments: ListDeploymentWithProcessInfo[];
    selectedDeploymentId: string | undefined;
    setSortOption: UseURLSortResult['setSortOption'];
};

function RiskTable({ currentDeployments, selectedDeploymentId, setSortOption }: RiskTableProps) {
    function onFetchData(state) {
        const newSortOption = convertTableSortToURLSetterSort(state);
        if (!newSortOption) {
            return;
        }
        setSortOption(newSortOption);
    }

    if (!currentDeployments.length) {
        return <NoResultsMessage message="No results found. Please refine your search." />;
    }
    return (
        <Table
            idAttribute="deployment.id"
            rows={currentDeployments}
            columns={riskTableColumnDescriptors}
            selectedRowId={selectedDeploymentId}
            onFetchData={onFetchData}
            noDataText="No results found. Please refine your search."
        />
    );
}

export default RiskTable;
