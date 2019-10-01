import React from 'react';
import PropTypes from 'prop-types';

import NoResultsMessage from 'Components/NoResultsMessage';
import CheckboxTable from 'Components/CheckboxTableV2';

import getColumns from './tableColumnDescriptors';

// Reads the table's state and chooses a sort option based on the sort column and direction.
function createSortOptionFromTableState(columns, state) {
    let sortOption;
    if (state.sorted.length && state.sorted[0].id) {
        // Column is selected for sorting.
        const column = columns.find(col => col.accessor === state.sorted[0].id);
        sortOption = {
            field: column.searchField,
            reversed: state.sorted[0].desc
        };
    } else {
        // Default to sorting by the Time column.
        sortOption = {
            field: columns.find(c => c.accessor === 'time').searchField,
            reversed: true
        };
    }
    return sortOption;
}

function ViolationsTable({
    violations,
    selectedAlertId,
    setSelectedAlertId,
    selectedRows,
    setSelectedRows,
    setSortOption
}) {
    if (!violations.length)
        return <NoResultsMessage message="No results found. Please refine your search." />;

    // Add a single row to the selected rows.
    function toggleRow(idtoToggle) {
        if (!selectedRows.find(id => id === idtoToggle)) {
            setSelectedRows(selectedRows.concat([idtoToggle]));
        } else {
            setSelectedRows(selectedRows.filter(id => id !== idtoToggle));
        }
    }

    // Calculate if all values on the current page are selected.
    const selectedIds = new Set(selectedRows);
    const allSelected = violations.reduce((cumm, curr) => {
        return cumm && selectedIds.has(curr.id);
    }, true);

    // Toggle rows either all on or all off for the current page.
    function toggleSelectAll() {
        if (allSelected) {
            setSelectedRows([]);
        } else {
            setSelectedRows(violations.map(violation => violation.id));
        }
    }

    // Select a single row to view in the side panel.
    function selectRow(alert) {
        setSelectedAlertId(alert.id);
    }

    const columns = getColumns(setSelectedAlertId);

    // Use the table's 'onFetchData' prop to set our sort option.
    function setSortOptionOnFetch(state) {
        setSortOption(createSortOptionFromTableState(columns, state));
    }

    return (
        <CheckboxTable
            rows={violations}
            columns={columns}
            onRowClick={selectRow}
            toggleRow={toggleRow}
            toggleSelectAll={toggleSelectAll}
            selection={selectedRows}
            selectedRowId={selectedAlertId}
            noDataText="No results found. Please refine your search."
            onFetchData={setSortOptionOnFetch}
        />
    );
}

ViolationsTable.propTypes = {
    violations: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    selectedAlertId: PropTypes.string,
    setSelectedAlertId: PropTypes.func.isRequired,
    selectedRows: PropTypes.arrayOf(PropTypes.string),
    setSelectedRows: PropTypes.func.isRequired,
    setSortOption: PropTypes.func.isRequired
};

ViolationsTable.defaultProps = {
    selectedAlertId: undefined,
    selectedRows: []
};

export default ViolationsTable;
