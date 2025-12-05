import NoResultsMessage from 'Components/NoResultsMessage';
import Table from 'Components/TableV2';

import riskTableColumnDescriptors from './riskTableColumnDescriptors';
import type { ListDeploymentWithProcessInfo } from 'services/DeploymentsService';
import type { ApiSortOptionSingle } from 'types/search';

function sortOptionFromTableState(state) {
    let sortOption: ApiSortOptionSingle | null = null;
    if (state.sorted.length && state.sorted[0].id) {
        const column = riskTableColumnDescriptors.find(
            (col) => col.accessor === state.sorted[0].id
        );
        sortOption = {
            // TODO we should be able to assert that column.searchField is not undefined after migrating away
            // from the legacy TableV2 and descriptor pattern
            field: column?.searchField ?? '',
            reversed: state.sorted[0].desc,
        };
    }
    return sortOption;
}

type RiskTableProps = {
    currentDeployments: ListDeploymentWithProcessInfo[];
    selectedDeploymentId: string | undefined;
    setSortOption: (sortOption: ApiSortOptionSingle) => void;
};

function RiskTable({ currentDeployments, selectedDeploymentId, setSortOption }: RiskTableProps) {
    function onFetchData(state) {
        const newSortOption = sortOptionFromTableState(state);
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
