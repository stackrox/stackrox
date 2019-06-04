import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { CLUSTERS_QUERY as QUERY } from 'queries/cluster';

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
        Cell: ({ original }) => {
            const { alerts } = original;
            if (!alerts.length) return 'No alerts';
            return (
                <span className="bg-alert-200 border border-alert-400 px-2 rounded text-alert-800">
                    {alerts.length} Alerts
                </span>
            );
        }
    },
    {
        Header: `Service Accounts`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        Cell: ({ original }) => {
            const { length } = original.serviceAccounts;
            if (!length) {
                return (
                    <span className="bg-alert-200 border border-alert-400 px-2 rounded text-alert-800">
                        No Matches
                    </span>
                );
            }
            if (length > 1) return `${length} Matches`;
            return original.serviceAccounts[0].name;
        }
    },
    {
        Header: `Roles`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        Cell: ({ original }) => {
            const { length } = original.k8sroles;
            if (!length) {
                return (
                    <span className="bg-alert-200 border border-alert-400 px-2 rounded text-alert-800">
                        No Matches
                    </span>
                );
            }
            if (length > 1) return `${length} Matches`;
            return original.k8sroles[0].name;
        }
    }
];

const createTableRows = data => data.results;

const Clusters = ({ onRowClick }) => (
    <List
        query={QUERY}
        entityType={entityTypes.CLUSTER}
        tableColumns={tableColumns}
        createTableRows={createTableRows}
        onRowClick={onRowClick}
    />
);

Clusters.propTypes = {
    onRowClick: PropTypes.func.isRequired
};

export default Clusters;
