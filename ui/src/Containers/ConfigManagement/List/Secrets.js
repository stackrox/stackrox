import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { SECRETS as QUERY } from 'queries/secret';
import uniq from 'lodash/uniq';
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
        Header: `Secret`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'name'
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
        Header: `File Types`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'files',
        // eslint-disable-next-line
        Cell: ({ original }) => {
            const { files } = original;
            if (!files.length) return 'No File Types';
            return (
                <span className="capitalize">
                    {uniq(files.map(file => file.type))
                        .join(', ')
                        .replace(/_/g, ' ')
                        .toLowerCase()}
                </span>
            );
        }
    },
    {
        Header: `Namespace`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'namespace'
    },
    {
        Header: `Deployments`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'deployments',
        Cell: ({ original }) => {
            const { deployments } = original;
            if (!deployments.length) return 'No Deployments';
            if (deployments.length === 1) return deployments[0].name;
            return `${deployments.length} matches`;
        }
    }
];

const createTableRows = data => data.secrets;

const Secrets = ({ onRowClick }) => (
    <List
        query={QUERY}
        entityType={entityTypes.SECRET}
        tableColumns={tableColumns}
        createTableRows={createTableRows}
        onRowClick={onRowClick}
    />
);

Secrets.propTypes = {
    onRowClick: PropTypes.func.isRequired
};

export default Secrets;
