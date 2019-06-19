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
        accessor: 'metadata.id'
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

const Namespaces = ({ className, selectedRowId, onRowClick }) => (
    <List
        className={className}
        query={QUERY}
        entityType={entityTypes.NAMESPACE}
        tableColumns={tableColumns}
        createTableRows={createTableRows}
        onRowClick={onRowClick}
        selectedRowId={selectedRowId}
        idAttribute="metadata.id"
    />
);

Namespaces.propTypes = {
    className: PropTypes.string,
    selectedRowId: PropTypes.string,
    onRowClick: PropTypes.func.isRequired
};

Namespaces.defaultProps = {
    className: '',
    selectedRowId: null
};

export default Namespaces;
