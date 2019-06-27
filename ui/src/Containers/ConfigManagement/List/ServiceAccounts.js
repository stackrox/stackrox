import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { SERVICE_ACCOUNTS as QUERY } from 'queries/serviceAccount';
import URLService from 'modules/URLService';

import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
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
            Header: `Service Accounts`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'name'
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
            Header: `Namespace`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'namespace',
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const {
                    id,
                    saNamespace: { metadata }
                } = original;
                if (!metadata) return 'No Matches';
                const { name, id: namespaceId } = metadata;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.NAMESPACE, namespaceId)
                    .url();
                return <TableCellLink pdf={pdf} url={url} text={name} />;
            }
        },
        {
            Header: `Roles`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { id, roles } = original;
                const { length } = roles;
                if (!length) return 'No Matches';
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.ROLE)
                    .url();
                if (length > 1)
                    return <TableCellLink pdf={pdf} url={url} text={`${length} Matches`} />;
                return original.roles[0].name;
            }
        }
    ];
    return tableColumns;
};

const createTableRows = data =>
    data.clusters.reduce((acc, curr) => [...acc, ...curr.serviceAccounts], []);

const ServiceAccounts = ({ match, location, className, selectedRowId, onRowClick }) => {
    const tableColumns = buildTableColumns(match, location);
    return (
        <List
            className={className}
            query={QUERY}
            entityType={entityTypes.SERVICE_ACCOUNT}
            tableColumns={tableColumns}
            createTableRows={createTableRows}
            onRowClick={onRowClick}
            selectedRowId={selectedRowId}
            idAttribute="id"
        />
    );
};

ServiceAccounts.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    className: PropTypes.string,
    selectedRowId: PropTypes.string,
    onRowClick: PropTypes.func.isRequired
};

ServiceAccounts.defaultProps = {
    className: '',
    selectedRowId: null
};

export default ServiceAccounts;
