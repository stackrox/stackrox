import React, { useContext } from 'react';
import pluralize from 'pluralize';

import entityTypes from 'constants/entityTypes';
import { DEPLOYMENTS_QUERY as QUERY } from 'queries/deployment';
import URLService from 'modules/URLService';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import { CLIENT_SIDE_SEARCH_OPTIONS as SEARCH_OPTIONS } from 'constants/searchOptions';
import searchContext from 'Containers/searchContext';
import queryService from 'modules/queryService';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import StatusChip from 'Components/StatusChip';
import List from './List';
import TableCellLink from './Link';
import filterByPolicyStatus from './utilities/filterByPolicyStatus';

const buildTableColumns = (match, location, entityContext) => {
    const tableColumns = [
        {
            Header: 'Id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'id'
        },
        {
            Header: `Deployment`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'name'
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
                  headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                  className: `w-1/8 ${defaultColumnClassName}`,
                  accessor: 'namespace',
                  // eslint-disable-next-line
                  Cell: ({ original, pdf }) => {
                      const { namespace, namespaceId, id } = original;
                      const url = URLService.getURL(match, location)
                          .push(id)
                          .push(entityTypes.NAMESPACE, namespaceId)
                          .url();
                      return <TableCellLink pdf={pdf} url={url} text={namespace} />;
                  }
              },
        {
            Header: `Policy Status`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { policyStatus } = original;
                return <StatusChip status={policyStatus} asString={pdf} />;
            },
            id: 'policyStatus',
            accessor: 'policyStatus'
        },
        {
            Header: `Images`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { imageCount, id } = original;
                if (imageCount === 0) return 'No images';
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.IMAGE)
                    .url();
                return (
                    <TableCellLink
                        pdf={pdf}
                        url={url}
                        text={`${imageCount} ${pluralize('image', imageCount)}`}
                    />
                );
            },
            accessor: 'imageCount'
        },
        {
            Header: `Secrets`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { secretCount, id } = original;
                if (secretCount === 0) return 'No secrets';
                const url = URLService.getURL(match, location)
                    .push(id)
                    .push(entityTypes.SECRET)
                    .url();
                return (
                    <TableCellLink
                        pdf={pdf}
                        url={url}
                        text={`${secretCount} ${pluralize('secret', secretCount)}`}
                    />
                );
            },
            accessor: 'secretCount'
        },
        entityContext && entityContext[entityTypes.SERVICE_ACCOUNT]
            ? null
            : {
                  Header: `Service Account`,
                  headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                  className: `w-1/8 ${defaultColumnClassName}`,
                  accessor: 'serviceAccount',
                  // eslint-disable-next-line
                  Cell: ({ original, pdf }) => {
                      const { serviceAccount, serviceAccountID, id } = original;
                      const url = URLService.getURL(match, location)
                          .push(id)
                          .push(entityTypes.SERVICE_ACCOUNT, serviceAccountID)
                          .url();
                      return <TableCellLink pdf={pdf} url={url} text={serviceAccount} />;
                  }
              }
    ];
    return tableColumns.filter(col => col);
};

const createTableRows = data => data.results;

const Deployments = ({
    match,
    location,
    className,
    selectedRowId,
    onRowClick,
    query,
    data,
    entityContext
}) => {
    const searchParam = useContext(searchContext);

    const autoFocusSearchInput = !selectedRowId;

    const tableColumns = buildTableColumns(match, location, entityContext);
    const {
        [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus,
        ...restQuery
    } = queryService.getQueryBasedOnSearchContext(query, searchParam);
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
            query={QUERY}
            variables={variables}
            entityType={entityTypes.DEPLOYMENT}
            tableColumns={tableColumns}
            createTableRows={createTableRowsFilteredByPolicyStatus}
            onRowClick={onRowClick}
            selectedRowId={selectedRowId}
            idAttribute="id"
            defaultSorted={[
                {
                    id: 'failingPolicies',
                    desc: true
                },
                {
                    id: 'name',
                    desc: false
                }
            ]}
            defaultSearchOptions={[SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]}
            data={filterByPolicyStatus(data, policyStatus)}
            autoFocusSearchInput={autoFocusSearchInput}
        />
    );
};
Deployments.propTypes = entityListPropTypes;
Deployments.defaultProps = entityListDefaultprops;

export default Deployments;
