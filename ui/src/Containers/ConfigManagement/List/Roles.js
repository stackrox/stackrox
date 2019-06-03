import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { K8S_ROLES as QUERY } from 'queries/role';
import { format } from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';

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
        Header: `Role`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'name'
    },
    {
        Header: `Type`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'type'
    },
    {
        Header: `Permissions`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        // eslint-disable-next-line
        Cell: ({ original }) => {
            const { verbs: permissions } = original;
            if (!permissions.length) return 'No Permissions';
            return <div className="capitalize">{permissions.join(', ')}</div>;
        }
    },
    {
        Header: `Created`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        Cell: ({ original }) => {
            const { createdAt } = original;
            return format(createdAt, dateTimeFormat);
        }
    },
    {
        Header: `Namespace Scope`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        Cell: ({ original }) => {
            const { roleNamespace } = original;
            if (!roleNamespace) return 'Cluster-wide';
            return roleNamespace.metadata.name;
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
    }
];

const createTableRows = data => data.clusters.reduce((acc, curr) => [...acc, ...curr.k8sroles], []);

const Roles = ({ onRowClick }) => (
    <List
        query={QUERY}
        entityType={entityTypes.ROLE}
        tableColumns={tableColumns}
        createTableRows={createTableRows}
        onRowClick={onRowClick}
    />
);

Roles.propTypes = {
    onRowClick: PropTypes.func.isRequired
};

export default Roles;
