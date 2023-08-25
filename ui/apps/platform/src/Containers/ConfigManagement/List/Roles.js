import React from 'react';
import pluralize from 'pluralize';
import { format } from 'date-fns';

import {
    defaultHeaderClassName,
    defaultColumnClassName,
    nonSortableHeaderClassName,
} from 'Components/Table';
import TableCellLink from 'Components/TableCellLink';
import dateTimeFormat from 'constants/dateTimeFormat';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import entityTypes from 'constants/entityTypes';
import { roleSortFields } from 'constants/sortFields';
import { K8S_ROLES_QUERY } from 'queries/role';
import queryService from 'utils/queryService';
import URLService from 'utils/URLService';
import List from './List';
import NoEntitiesIconText from './utilities/NoEntitiesIconText';

export const defaultRoleSort = [
    {
        id: roleSortFields.ROLE,
        desc: false,
    },
];

const buildTableColumns = (match, location, entityContext) => {
    const tableColumns = [
        {
            Header: 'Id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'id',
        },
        {
            Header: `Role`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'name',
            id: roleSortFields.ROLE,
            sortField: roleSortFields.ROLE,
        },
        {
            Header: `Type`,
            headerClassName: `w-1/8 ${nonSortableHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'type',
            sortable: false,
        },
        {
            Header: `Permissions`,
            headerClassName: `w-1/8 ${nonSortableHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { verbs: permissions } = original;
                if (!permissions.length) {
                    return 'No Permissions';
                }
                return <div className="capitalize">{permissions.join(', ')}</div>;
            },
            accessor: 'verbs',
            sortable: false,
        },
        {
            Header: `Created`,
            headerClassName: `w-1/8 ${nonSortableHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { createdAt } = original;
                return format(createdAt, dateTimeFormat);
            },
            accessor: 'createdAt',
            sortable: false,
        },
        entityContext && entityContext[entityTypes.CLUSTER]
            ? null
            : {
                  Header: `Cluster`,
                  headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                  className: `w-1/8 ${defaultColumnClassName}`,
                  accessor: 'clusterName',
                  Cell: ({ original, pdf }) => {
                      const { clusterName, clusterId, id } = original;
                      const url = URLService.getURL(match, location)
                          .push(id)
                          .push(entityTypes.CLUSTER, clusterId)
                          .url();
                      return (
                          <TableCellLink pdf={pdf} url={url}>
                              {clusterName}
                          </TableCellLink>
                      );
                  },
                  id: roleSortFields.CLUSTER,
                  sortField: roleSortFields.CLUSTER,
              },
        {
            Header: `Namespace Scope`,
            headerClassName: `w-1/8 ${nonSortableHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { roleNamespace, id } = original;
                if (!roleNamespace) {
                    return 'Cluster-wide';
                }
                const {
                    metadata: { name, id: namespaceId },
                } = roleNamespace;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.NAMESPACE, namespaceId)
                    .url();
                return (
                    <TableCellLink pdf={pdf} url={url}>
                        {name}
                    </TableCellLink>
                );
            },
            accessor: 'roleNamespace.metadata.name',
            sortable: false,
        },
        {
            Header: `Users & Groups`,
            headerClassName: `w-1/8 ${nonSortableHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { serviceAccounts, subjects } = original;
                const { length: serviceAccountsLength } = serviceAccounts;
                const { length: subjectsLength } = subjects;
                if (!subjectsLength) {
                    return !serviceAccountsLength ||
                        (serviceAccountsLength === 1 && serviceAccounts[0].message) ? (
                        <NoEntitiesIconText text="No Users & Groups" isTextOnly={pdf} />
                    ) : (
                        'No Users & Groups'
                    );
                }
                const url = URLService.getURL(match, location)
                    .push(original.id)
                    .push(entityTypes.SUBJECT)
                    .url();
                const text = `${subjectsLength} ${pluralize('Users & Groups', subjectsLength)}`;
                if (subjectsLength > 1) {
                    return (
                        <TableCellLink pdf={pdf} url={url}>
                            {text}
                        </TableCellLink>
                    );
                }
                const subject = subjects[0];
                return (
                    <TableCellLink pdf={pdf} url={url}>
                        {subject.name}
                    </TableCellLink>
                );
            },
            id: 'subjects',
            accessor: (d) => d.subjects,
            sortable: false,
        },
        {
            Header: `Service Accounts`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { serviceAccounts, subjects, id } = original;
                const { length: serviceAccountsLength } = serviceAccounts;
                const { length: subjectsLength } = subjects;
                if (
                    (!serviceAccountsLength ||
                        (serviceAccountsLength === 1 && serviceAccounts[0].message)) &&
                    !subjectsLength
                ) {
                    return <NoEntitiesIconText text="No Service Accounts" isTextOnly={pdf} />;
                }
                if (!serviceAccountsLength) {
                    return 'No Service Accounts';
                }
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.SERVICE_ACCOUNT)
                    .url();
                const text = `${serviceAccountsLength} ${pluralize(
                    'Service Accounts',
                    serviceAccountsLength
                )}`;
                if (serviceAccountsLength > 1) {
                    return (
                        <TableCellLink pdf={pdf} url={url}>
                            {text}
                        </TableCellLink>
                    );
                }
                const serviceAccount = serviceAccounts[0];
                return (
                    <TableCellLink pdf={pdf} url={url}>
                        {serviceAccount.name}
                    </TableCellLink>
                );
            },
            accessor: 'serviceAccounts',
            sortable: false,
        },
    ];
    return tableColumns.filter((col) => col);
};

const createTableRows = (data) => data.results;

const Roles = ({
    match,
    location,
    className,
    selectedRowId,
    onRowClick,
    query,
    data,
    totalResults,
    entityContext,
}) => {
    const autoFocusSearchInput = !selectedRowId;
    const tableColumns = buildTableColumns(match, location, entityContext);
    const queryText = queryService.objectToWhereClause(query);
    const variables = queryText ? { query: queryText } : null;
    return (
        <List
            className={className}
            query={K8S_ROLES_QUERY}
            variables={variables}
            entityType={entityTypes.ROLE}
            tableColumns={tableColumns}
            createTableRows={createTableRows}
            onRowClick={onRowClick}
            selectedRowId={selectedRowId}
            idAttribute="id"
            defaultSorted={defaultRoleSort}
            data={data}
            totalResults={totalResults}
            autoFocusSearchInput={autoFocusSearchInput}
        />
    );
};
Roles.propTypes = entityListPropTypes;
Roles.defaultProps = entityListDefaultprops;

export default Roles;
