import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { CLUSTERS_QUERY as QUERY } from 'queries/cluster';

import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import LabelChip from 'Components/LabelChip';
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
            return <LabelChip text={`${alerts.length} Alerts`} type="alert" />;
        }
    },
    {
        Header: `Service Accounts`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        Cell: ({ original }) => {
            const { length } = original.serviceAccounts;
            if (!length) {
                return <LabelChip text="No Matches" type="alert" />;
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
                return <LabelChip text="No Matches" type="alert" />;
            }
            if (length > 1) return `${length} Matches`;
            return original.k8sroles[0].name;
        }
    }
];

const createTableRows = data => data.results;

const Clusters = ({ className, selectedRowId, onRowClick }) => (
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

Clusters.propTypes = {
    className: PropTypes.string,
    selectedRowId: PropTypes.string,
    onRowClick: PropTypes.func.isRequired
};

Clusters.defaultProps = {
    className: '',
    selectedRowId: null
};

export default Clusters;
