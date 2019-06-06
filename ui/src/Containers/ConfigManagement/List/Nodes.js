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

const Nodes = ({ onRowClick }) => (
    <List
        query={QUERY}
        entityType={entityTypes.NODE}
        tableColumns={tableColumns}
        createTableRows={createTableRows}
        onRowClick={onRowClick}
    />
);

Nodes.propTypes = {
    onRowClick: PropTypes.func.isRequired
};

export default Nodes;
