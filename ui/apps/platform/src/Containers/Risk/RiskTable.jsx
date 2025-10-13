import React from 'react';
import PropTypes from 'prop-types';

import NoResultsMessage from 'Components/NoResultsMessage';
import Table from 'Components/TableV2';

import riskTableColumnDescriptors from './riskTableColumnDescriptors';

function sortOptionFromTableState(state) {
    let sortOption = null;
    if (state.sorted.length && state.sorted[0].id) {
        const column = riskTableColumnDescriptors.find(
            (col) => col.accessor === state.sorted[0].id
        );
        sortOption = {
            field: column.searchField,
            reversed: state.sorted[0].desc,
        };
    }
    return sortOption;
}

function RiskTable({
    currentDeployments,
    setSelectedDeploymentId,
    selectedDeploymentId,
    setSortOption,
}) {
    function onFetchData(state) {
        const newSortOption = sortOptionFromTableState(state);
        if (!newSortOption) {
            return;
        }
        setSortOption(newSortOption);
    }

    function updateSelectedDeployment({ deployment }) {
        setSelectedDeploymentId(deployment.id);
    }

    if (!currentDeployments.length) {
        return <NoResultsMessage message="No results found. Please refine your search." />;
    }
    return (
        <Table
            idAttribute="deployment.id"
            rows={currentDeployments}
            columns={riskTableColumnDescriptors}
            onRowClick={updateSelectedDeployment}
            selectedRowId={selectedDeploymentId}
            onFetchData={onFetchData}
            noDataText="No results found. Please refine your search."
        />
    );
}

RiskTable.propTypes = {
    currentDeployments: PropTypes.arrayOf(PropTypes.object).isRequired,
    selectedDeploymentId: PropTypes.string,
    setSelectedDeploymentId: PropTypes.func.isRequired,
    setSortOption: PropTypes.func.isRequired,
};

RiskTable.defaultProps = {
    selectedDeploymentId: undefined,
};

export default RiskTable;
