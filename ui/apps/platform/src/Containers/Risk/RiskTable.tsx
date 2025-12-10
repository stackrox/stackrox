import NoResultsMessage from 'Components/NoResultsMessage';
import Table from 'Components/TableV2';

import type { ListDeploymentWithProcessInfo } from 'services/DeploymentsService';
import type { UseURLSortResult } from 'hooks/useURLSort';
import type { SortOption } from 'types/table';

import riskTableColumnDescriptors from './riskTableColumnDescriptors';

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
