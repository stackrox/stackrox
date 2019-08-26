import React from 'react';
import PropTypes from 'prop-types';

import NoResultsMessage from 'Components/NoResultsMessage';
import Table from 'Components/TableV2';

import columns from './tableColumnDescriptor';

function sortOptionFromTableState(state) {
    let sortOption;
    if (state.sorted.length && state.sorted[0].id) {
        const column = columns.find(col => col.accessor === state.sorted[0].id);
        sortOption = {
            field: column.searchField,
            reversed: state.sorted[0].desc
        };
    } else {
        sortOption = {
            field: 'Priority',
            reversed: false
        };
    }
    return sortOption;
}

function RiskTable({
    currentDeployments,
    setSelectedDeploymentId,
    selectedDeploymentId,
    setSortOption
}) {
    function onFetchData(state) {
        setSortOption(sortOptionFromTableState(state));
    }

    function updateSelectedDeployment({ deployment }) {
        setSelectedDeploymentId(deployment.id);
    }

    if (!currentDeployments.length)
        return <NoResultsMessage message="No results found. Please refine your search." />;
    return (
        <Table
            idAttribute="deployment.id"
            rows={currentDeployments}
            columns={columns}
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
    setSortOption: PropTypes.func.isRequired
};

RiskTable.defaultProps = {
    selectedDeploymentId: undefined
};

export default RiskTable;
