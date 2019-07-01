import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { ALL_NAMESPACES as QUERY } from 'queries/namespace';
import URLService from 'modules/URLService';

import { sortValueByLength } from 'sorters/sorters';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import List from './List';
import TableCellLink from './Link';

const buildTableColumns = (match, location) => {
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
            accessor: 'metadata.clusterName',
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { metadata } = original;
                if (!metadata) return '-';
                const { clusterName, clusterId, id } = metadata;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.CLUSTER, clusterId)
                    .url();
                return <TableCellLink pdf={pdf} url={url} text={clusterName} />;
            }
        },
        {
            Header: `Secrets`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { numSecrets, metadata } = original;
                if (!metadata || numSecrets === 0) return 'No matches';
                const { id } = metadata;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.SECRET)
                    .url();
                return <TableCellLink pdf={pdf} url={url} text={`${numSecrets} matches`} />;
            },
            id: 'numSecrets',
            accessor: d => d.numSecrets,
            sortMethod: sortValueByLength
        }
    ];
    return tableColumns;
};

const createTableRows = data => data.results;

const Namespaces = ({ match, location, className, selectedRowId, onRowClick }) => {
    const tableColumns = buildTableColumns(match, location);
    return (
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
};

Namespaces.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    className: PropTypes.string,
    selectedRowId: PropTypes.string,
    onRowClick: PropTypes.func.isRequired
};

Namespaces.defaultProps = {
    className: '',
    selectedRowId: null
};

export default Namespaces;
