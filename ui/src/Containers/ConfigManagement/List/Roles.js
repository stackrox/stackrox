import React from 'react';
import entityTypes from 'constants/entityTypes';
import { K8S_ROLES as QUERY } from 'queries/role';
import { format } from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';
import URLService from 'modules/URLService';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import { sortValueByLength, sortDate } from 'sorters/sorters';
import queryService from 'modules/queryService';
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
            },
            accessor: 'verbs',
            sortMethod: sortValueByLength
        },
        {
            Header: `Created`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { createdAt } = original;
                return format(createdAt, dateTimeFormat);
            },
            accessor: 'createdAt',
            sortMethod: sortDate
        },
        {
            Header: `Namespace Scope`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { roleNamespace, id } = original;
                if (!roleNamespace) return 'Cluster-wide';
                const {
                    metadata: { name, id: namespaceId }
                } = roleNamespace;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.NAMESPACE, namespaceId)
                    .url();
                return <TableCellLink pdf={pdf} url={url} text={name} />;
            },
            accessor: 'roleNamespace.metadata.name'
        },
        {
            Header: `Service Accounts`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { serviceAccounts, id } = original;
                const { length } = serviceAccounts;
                if (!length) return <LabelChip text="No Matches" type="alert" />;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.SERVICE_ACCOUNT)
                    .url();
                if (length > 1)
                    return <TableCellLink pdf={pdf} url={url} text={`${length} Matches`} />;
                const serviceAccount = serviceAccounts[0];
                if (serviceAccount.name) return serviceAccount.name;
                return <LabelChip text={serviceAccount.message} type="alert" />;
            },
            accessor: 'serviceAccounts',
            sortMethod: sortValueByLength
        }
    ];
    return tableColumns;
};

const createTableRows = data => data.results;

const Roles = ({ match, location, className, selectedRowId, onRowClick, query }) => {
    const tableColumns = buildTableColumns(match, location);
    const queryText = queryService.objectToWhereClause(query);
    const variables = queryText ? { query: queryText } : null;
    return (
        <List
            className={className}
            query={QUERY}
            variables={variables}
            entityType={entityTypes.ROLE}
            tableColumns={tableColumns}
            createTableRows={createTableRows}
            onRowClick={onRowClick}
            selectedRowId={selectedRowId}
            idAttribute="id"
        />
    );
};
Roles.propTypes = entityListPropTypes;
Roles.defaultProps = entityListDefaultprops;

export default Roles;
