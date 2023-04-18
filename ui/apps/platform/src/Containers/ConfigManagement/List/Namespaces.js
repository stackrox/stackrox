import React, { useContext } from 'react';
import pluralize from 'pluralize';

import PolicyStatusIconText from 'Components/PatternFly/IconText/PolicyStatusIconText';
import {
    defaultHeaderClassName,
    defaultColumnClassName,
    nonSortableHeaderClassName,
} from 'Components/Table';
import TableCellLink from 'Components/TableCellLink';
import searchContext from 'Containers/searchContext';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import entityTypes from 'constants/entityTypes';
import { CLIENT_SIDE_SEARCH_OPTIONS as SEARCH_OPTIONS } from 'constants/searchOptions';
import { namespaceSortFields } from 'constants/sortFields';
import { NAMESPACES_NO_POLICIES_QUERY } from 'queries/namespace';
import queryService from 'utils/queryService';
import URLService from 'utils/URLService';
import List from './List';

import filterByPolicyStatus from './utilities/filterByPolicyStatus';

export const defaultNamespaceSort = [
    {
        id: namespaceSortFields.NAMESPACE,
        desc: false,
    },
];

const buildTableColumns = (match, location, entityContext) => {
    const tableColumns = [
        {
            Header: 'Id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'metadata.id',
        },
        {
            Header: `Namespace`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'metadata.name',
            id: namespaceSortFields.NAMESPACE,
            sortField: namespaceSortFields.NAMESPACE,
        },
        entityContext && entityContext[entityTypes.CLUSTER]
            ? null
            : {
                  Header: `Cluster`,
                  headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                  className: `w-1/8 ${defaultColumnClassName}`,
                  accessor: 'metadata.clusterName',
                  Cell: ({ original, pdf }) => {
                      const { metadata } = original;
                      if (!metadata) {
                          return '-';
                      }
                      const { clusterName, clusterId, id } = metadata;
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
                  id: namespaceSortFields.CLUSTER,
                  sortField: namespaceSortFields.CLUSTER,
              },
        {
            Header: `Policy Status`,
            headerClassName: `w-1/8 ${nonSortableHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const {
                    policyStatus: { status },
                } = original;
                return <PolicyStatusIconText isPass={status === 'pass'} isTextOnly={pdf} />;
            },
            id: 'status',
            accessor: (d) => d.policyStatus.status,
            sortable: false,
        },
        {
            Header: `Secrets`,
            headerClassName: `w-1/8 ${nonSortableHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { numSecrets, metadata } = original;
                if (!metadata || numSecrets === 0) {
                    return 'No Secrets';
                }
                const { id } = metadata;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.SECRET)
                    .url();
                const text = `${numSecrets} ${pluralize('Secrets', numSecrets)}`;
                return (
                    <TableCellLink pdf={pdf} url={url}>
                        {text}
                    </TableCellLink>
                );
            },
            id: 'numSecrets',
            accessor: (d) => d.numSecrets,
            sortable: false,
        },
        {
            Header: `Users & Groups`,
            headerClassName: `w-1/8 ${nonSortableHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { subjectsCount, metadata } = original;
                if (!subjectsCount || subjectsCount === 0) {
                    return 'No Users & Groups';
                }
                const { id } = metadata;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.SUBJECT)
                    .url();
                const text = `${subjectsCount} ${pluralize('Users & Groups', subjectsCount)}`;
                return (
                    <TableCellLink pdf={pdf} url={url}>
                        {text}
                    </TableCellLink>
                );
            },
            accessor: 'subjectCount',
            sortable: false,
        },
        {
            Header: `Service Accounts`,
            headerClassName: `w-1/8 ${nonSortableHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { serviceAccountCount, metadata } = original;
                if (!serviceAccountCount || serviceAccountCount === 0) {
                    return 'No Service Accounts';
                }
                const { id } = metadata;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.SERVICE_ACCOUNT)
                    .url();
                const text = `${serviceAccountCount} ${pluralize(
                    'Service Accounts',
                    serviceAccountCount
                )}`;
                return (
                    <TableCellLink pdf={pdf} url={url}>
                        {text}
                    </TableCellLink>
                );
            },
            accessor: 'serviceAccountCount',
            sortable: false,
        },
        {
            Header: `Roles`,
            headerClassName: `w-1/8 ${nonSortableHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { k8sRoleCount, metadata } = original;
                if (!k8sRoleCount || k8sRoleCount === 0) {
                    return 'No Roles';
                }
                const { id } = metadata;
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.ROLE)
                    .url();
                const text = `${k8sRoleCount} ${pluralize('Roles', k8sRoleCount)}`;
                return (
                    <TableCellLink pdf={pdf} url={url}>
                        {text}
                    </TableCellLink>
                );
            },
            accessor: 'k8sRoleCount',
            sortable: false,
        },
    ];
    return tableColumns.filter((col) => col);
};

const createTableRows = (data) => data.results;

const Namespaces = ({
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
    const searchParam = useContext(searchContext);

    const autoFocusSearchInput = !selectedRowId;
    const tableColumns = buildTableColumns(match, location, entityContext);
    const { [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus, ...restQuery } =
        queryService.getQueryBasedOnSearchContext(query, searchParam);
    const queryText = queryService.objectToWhereClause({ ...restQuery });
    const variables = queryText ? { query: queryText } : null;

    function createTableRowsFilteredByPolicyStatus(items) {
        const tableRows = createTableRows(items);
        const filteredTableRows = filterByPolicyStatus(tableRows, policyStatus);
        return filteredTableRows;
    }

    return (
        <List
            className={className}
            query={NAMESPACES_NO_POLICIES_QUERY}
            variables={variables}
            entityType={entityTypes.NAMESPACE}
            tableColumns={tableColumns}
            createTableRows={createTableRowsFilteredByPolicyStatus}
            onRowClick={onRowClick}
            selectedRowId={selectedRowId}
            idAttribute="metadata.id"
            defaultSorted={defaultNamespaceSort}
            defaultSearchOptions={[SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]}
            data={filterByPolicyStatus(data, policyStatus)}
            totalResults={totalResults}
            autoFocusSearchInput={autoFocusSearchInput}
        />
    );
};
Namespaces.propTypes = entityListPropTypes;
Namespaces.defaultProps = entityListDefaultprops;

export default Namespaces;
