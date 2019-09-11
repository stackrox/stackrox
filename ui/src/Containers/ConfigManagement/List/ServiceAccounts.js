import React from 'react';
import URLService from 'modules/URLService';
import entityTypes from 'constants/entityTypes';
import { SERVICE_ACCOUNTS as QUERY } from 'queries/serviceAccount';
import { sortValueByLength } from 'sorters/sorters';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import queryService from 'modules/queryService';
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
            Header: `Service Accounts`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            accessor: 'name'
        },
        {
            Header: `Cluster Admin Role`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { clusterAdmin } = original;
                return clusterAdmin ? 'Enabled' : 'Disabled';
            },
            accessor: 'clusterAdmin'
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
        entityContext && entityContext[entityTypes.NAMESPACE]
            ? null
            : {
                  Header: `Namespace`,
                  headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                  className: `w-1/10 ${defaultColumnClassName}`,
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
                if (!length) return 'No Roles';
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.ROLE)
                    .url();
                if (length > 1)
                    return (
                        <TableCellLink
                            pdf={pdf}
                            url={url}
                            text={`${length} ${pluralize('Roles', length)}`}
                        />
                    );
                return original.roles[0].name;
            },
            accessor: 'roles',
            sortMethod: sortValueByLength
        },
        {
            Header: `Deployments`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { id, deploymentCount } = original;
                if (!deploymentCount) return 'No Deployments';
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.DEPLOYMENT)
                    .url();
                return (
                    <TableCellLink
                        pdf={pdf}
                        url={url}
                        text={`${deploymentCount} ${pluralize('Deployment', deploymentCount)}`}
                    />
                );
            },
            accessor: 'deploymentCount'
        }
    ];
    return tableColumns.filter(col => col);
};

const createTableRows = data => data.results;

const ServiceAccounts = ({
    match,
    location,
    className,
    selectedRowId,
    onRowClick,
    query,
    data,
    entityContext
}) => {
    const autoFocusSearchInput = !selectedRowId;
    const tableColumns = buildTableColumns(match, location, entityContext);
    const queryText = queryService.objectToWhereClause(query);
    const variables = queryText ? { query: queryText } : null;
    return (
        <List
            className={className}
            query={QUERY}
            variables={variables}
            entityType={entityTypes.SERVICE_ACCOUNT}
            tableColumns={tableColumns}
            createTableRows={createTableRows}
            onRowClick={onRowClick}
            selectedRowId={selectedRowId}
            idAttribute="id"
            defaultSorted={[
                {
                    id: 'clusterAdmin',
                    desc: true
                },
                {
                    id: 'name',
                    desc: false
                }
            ]}
            data={data}
            autoFocusSearchInput={autoFocusSearchInput}
        />
    );
};
ServiceAccounts.propTypes = entityListPropTypes;
ServiceAccounts.defaultProps = entityListDefaultprops;

export default ServiceAccounts;
