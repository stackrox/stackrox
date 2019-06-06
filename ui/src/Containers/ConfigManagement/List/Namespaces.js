import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { ALL_NAMESPACES as QUERY } from 'queries/namespace';

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
        Header: `Namespace`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'metadata.name'
    },
    {
        Header: `Cluster`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'metadata.clusterName'
    },
    {
        Header: `Secrets`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        Cell: ({ original }) => {
            const { numSecrets } = original;
            if (numSecrets === 0) return 'No matches';
            return `${numSecrets} matches`;
        }
    }
];

const createTableRows = data => data.results;

const Namespaces = ({ onRowClick }) => (
    <List
        query={QUERY}
        entityType={entityTypes.NODE}
        tableColumns={tableColumns}
        createTableRows={createTableRows}
        onRowClick={onRowClick}
    />
);

Namespaces.propTypes = {
    onRowClick: PropTypes.func.isRequired
};

export default Namespaces;
