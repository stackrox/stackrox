import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { DEPLOYMENTS_QUERY as QUERY } from 'queries/deployment';

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
        Header: `Deployment`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'name'
    },
    {
        Header: `Cluster`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'clusterName'
    },
    {
        Header: `Namespace`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'namespace'
    },
    {
        Header: `Service Account`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'serviceAccount'
    },
    {
        Header: `Alerts`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'alerts',
        // eslint-disable-next-line
        Cell: ({ original }) => {
            const { alerts } = original;
            if (!alerts.length) return 'No alerts';
            return (
                <span className="bg-alert-200 border border-alert-400 px-2 rounded text-alert-800">
                    {alerts.length} Alerts
                </span>
            );
        }
    }
];

const createTableRows = data => data.results;

const Deployments = ({ onRowClick }) => (
    <List
        query={QUERY}
        entityType={entityTypes.DEPLOYMENT}
        tableColumns={tableColumns}
        createTableRows={createTableRows}
        onRowClick={onRowClick}
    />
);

Deployments.propTypes = {
    onRowClick: PropTypes.func.isRequired
};

export default Deployments;
