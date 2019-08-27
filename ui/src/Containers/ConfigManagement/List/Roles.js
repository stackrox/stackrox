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
import pluralize from 'pluralize';
import List from './List';
import TableCellLink from './Link';

const buildTableColumns = (match, location, entityContext) => {
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
        entityContext && entityContext[entityTypes.CLUSTER]
            ? null
            : {
                  Header: `Cluster`,
                  headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                  className: `w-1/8 ${defaultColumnClassName}`,
                  accessor: 'clusterName',
                  // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                      const { clusterName, clusterId, id } = original;
                      const url = URLService.getURL(match, location)
                          .push(id)
                          .push(entityTypes.CLUSTER, clusterId)
                          .url();
                      return <TableCellLink pdf={pdf} url={url} text={clusterName} />;
                  }
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
            Header: `Users & Groups`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { serviceAccounts, subjects } = original;
                const { length: serviceAccountsLength } = serviceAccounts;
                const { length: subjectsLength } = subjects;
                if (
                    (!serviceAccountsLength ||
                        (serviceAccountsLength === 1 && serviceAccounts[0].message)) &&
                    !subjectsLength
                ) {
                    return <LabelChip text="No Users & Groups" type="alert" />;
                }
                if (!subjectsLength) {
                    return 'No Users & Groups';
                }
                const url = URLService.getURL(match, location)
                    .push(original.id)
                    .push(entityTypes.SUBJECT)
                    .url();
                if (subjectsLength > 1)
                    return (
                        <TableCellLink
                            pdf={pdf}
                            url={url}
                            text={`${subjectsLength} ${pluralize(
                                'Users & Groups',
                                subjectsLength
                            )}`}
                        />
                    );
                const subject = subjects[0];
                return <TableCellLink pdf={pdf} url={url} text={subject.name} />;
            },
            id: 'subjects',
            accessor: d => d.subjects,
            sortMethod: sortValueByLength
        },
        {
            Header: `Service Accounts`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { serviceAccounts, subjects, id } = original;
                const { length: serviceAccountsLength } = serviceAccounts;
                const { length: subjectsLength } = subjects;
                if (
                    (!serviceAccountsLength ||
                        (serviceAccountsLength === 1 && serviceAccounts[0].message)) &&
                    !subjectsLength
                ) {
                    return <LabelChip text="No Service Accounts" type="alert" />;
                }
                if (!serviceAccountsLength) {
                    return 'No Service Accounts';
                }
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.SERVICE_ACCOUNT)
                    .url();
                if (serviceAccountsLength > 1)
                    return (
                        <TableCellLink
                            pdf={pdf}
                            url={url}
                            text={`${serviceAccountsLength} ${pluralize(
                                'Service Accounts',
                                serviceAccountsLength
                            )}`}
                        />
                    );
                const serviceAccount = serviceAccounts[0];
                return <TableCellLink pdf={pdf} url={url} text={serviceAccount.name} />;
            },
            accessor: 'serviceAccounts',
            sortMethod: sortValueByLength
        }
    ];
    return tableColumns.filter(col => col);
};

const createTableRows = data => data.results;

const Roles = ({
    match,
    location,
    className,
    selectedRowId,
    onRowClick,
    query,
    data,
    entityContext
}) => {
    const tableColumns = buildTableColumns(match, location, entityContext);
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
            data={data}
        />
    );
};
Roles.propTypes = entityListPropTypes;
Roles.defaultProps = entityListDefaultprops;

export default Roles;
