import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { NODES_QUERY as QUERY } from 'queries/node';

import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import List from './List';

const tableColumns = [
    {
        Header: 'Id',
        headerClassName: 'hidden',
        className: 'hidden',
        accessor: 'id'
    },
    {
        Header: `Node`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'name'
    },
    {
        Header: `Cluster`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'clusterName'
    }
];

const createTableRows = data => data.results.reduce((acc, curr) => [...acc, ...curr.nodes], []);

const Nodes = ({ className, selectedRowId, onRowClick }) => (
    <List
        className={className}
        query={QUERY}
        entityType={entityTypes.NODE}
        tableColumns={tableColumns}
        createTableRows={createTableRows}
        onRowClick={onRowClick}
        selectedRowId={selectedRowId}
        idAttribute="id"
    />
);

Nodes.propTypes = {
    className: PropTypes.string,
    selectedRowId: PropTypes.string,
    onRowClick: PropTypes.func.isRequired
};

Nodes.defaultProps = {
    className: '',
    selectedRowId: null
};

export default Nodes;
