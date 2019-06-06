import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import QUERY from 'queries/subject';

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
        Header: 'Users & Groups',
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'subject.name'
    },
    {
        Header: 'Type',
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'type'
    },
    {
        Header: `Permissions`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        Cell: ({ original }) => {
            const { scopedPermissions } = original;
            if (!scopedPermissions.length) return 'No Scoped Permissions';
            const result = scopedPermissions
                .map(({ scope, permissions }) => `${scope} (${permissions.length})`)
                .join(', ');
            return result;
        }
    },
    {
        Header: `Cluster Admin Role`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        Cell: ({ original }) => {
            const { clusterAdmin } = original;
            return clusterAdmin ? 'Enabled' : 'Disabled';
        }
    },
    {
        Header: `Roles`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        Cell: ({ original }) => {
            const { length } = original.roles;
            if (length === 0) return 'No Matches';
            if (length > 1) return `${length} Matches`;
            return original.roles[0].name;
        }
    }
];

const createTableRows = data => data.clusters.reduce((acc, curr) => [...acc, ...curr.subjects], []);

const Subjects = ({ onRowClick }) => (
    <List
        query={QUERY}
        entityType={entityTypes.SUBJECT}
        tableColumns={tableColumns}
        createTableRows={createTableRows}
        onRowClick={onRowClick}
    />
);

Subjects.propTypes = {
    onRowClick: PropTypes.func.isRequired
};

export default Subjects;
