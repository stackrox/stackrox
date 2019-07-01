import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { CLUSTERS_QUERY as QUERY } from 'queries/cluster';
import URLService from 'modules/URLService';

import { sortValueByLength } from 'sorters/sorters';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import LabelChip from 'Components/LabelChip';
import List from './List';
import TableCellLink from './Link';

const buildTableColumns = (match, location) => {
    const tableColumns = [
        {
            Header: 'Id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'id'
        },
        {
            Header: `Cluster`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'name'
        },
        {
            Header: `Policies Violated`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'alerts',
            // eslint-disable-next-line
            Cell: ({ original, pdf  }) => {
                const { alerts } = original;
                if (!alerts.length) return 'No alerts';
                return <LabelChip text={`${alerts.length} Alerts`} type="alert" />;
            }
        },
        {
            Header: `Service Accounts`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { length } = original.serviceAccounts;
                if (!length) {
                    return <LabelChip text="No Matches" type="alert" />;
                }
                const url = URLService.getURL(match, location)
                    .push(original.id)
                    .push(entityTypes.SERVICE_ACCOUNT)
                    .url();
                if (length > 1)
                    return <TableCellLink pdf={pdf} url={url} text={`${length} Matches`} />;
                return original.serviceAccounts[0].name;
            },
            id: 'serviceAccounts',
            accessor: d => d.serviceAccounts,
            sortMethod: sortValueByLength
        },
        {
            Header: `Roles`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { length } = original.k8sroles;
                if (!length) return <LabelChip text="No Matches" type="alert" />;
                const url = URLService.getURL(match, location)
                    .push(original.id)
                    .push(entityTypes.ROLE)
                    .url();
                if (length > 1)
                    return <TableCellLink pdf={pdf} url={url} text={`${length} Matches`} />;
                return original.k8sroles[0].name;
            },
            id: 'k8sroles',
            accessor: d => d.k8sroles,
            sortMethod: sortValueByLength
        }
    ];
    return tableColumns;
};

const createTableRows = data => data.results;

const Clusters = ({ match, location, className, selectedRowId, onRowClick }) => {
    const tableColumns = buildTableColumns(match, location);
    return (
        <List
            className={className}
            query={QUERY}
            entityType={entityTypes.CLUSTER}
            tableColumns={tableColumns}
            createTableRows={createTableRows}
            onRowClick={onRowClick}
            selectedRowId={selectedRowId}
            idAttribute="id"
        />
    );
};

Clusters.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    className: PropTypes.string,
    selectedRowId: PropTypes.string,
    onRowClick: PropTypes.func.isRequired
};

Clusters.defaultProps = {
    className: '',
    selectedRowId: null
};

export default Clusters;
