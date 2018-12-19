import React from 'react';
import PropTypes from 'prop-types';

import Table from 'Components/Table';

function List(props) {
    const { rows, selectRow, selectedListItem, selectedIdAttribute, columns } = props;
    if (!rows.length) return null;
    return (
        <Table
            columns={columns}
            rows={rows}
            onRowClick={selectRow}
            showThead={false}
            idAttribute={selectedIdAttribute}
            selectedRowId={selectedListItem[selectedIdAttribute]}
            noDataText="No Items Available. Create a new one below."
        />
    );
}

List.propTypes = {
    rows: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    columns: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    selectRow: PropTypes.func.isRequired,
    selectedListItem: PropTypes.shape({}),
    selectedIdAttribute: PropTypes.string.isRequired
};

List.defaultProps = {
    selectedListItem: null
};

export default List;
