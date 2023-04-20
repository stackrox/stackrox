import React from 'react';
import { withRouter } from 'react-router-dom';
import { gql } from '@apollo/client';
import { format } from 'date-fns';
import pluralize from 'pluralize';

import {
    defaultHeaderClassName,
    defaultColumnClassName,
    nonSortableHeaderClassName,
} from 'Components/Table';
import TableCellLink from 'Components/TableCellLink';
import dateTimeFormat from 'constants/dateTimeFormat';
import entityTypes from 'constants/entityTypes';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import { nodeSortFields } from 'constants/sortFields';
import queryService from 'utils/queryService';
import URLService from 'utils/URLService';
import List from './List';
import NoEntitiesIconText from './utilities/NoEntitiesIconText';

const QUERY = gql`
    query nodes($query: String, $pagination: Pagination) {
        results: nodes(query: $query, pagination: $pagination) {
            id
            name
            clusterName
            clusterId
            osImage
            containerRuntimeVersion
            joinedAt
            nodeComplianceControlCount(query: "Standard:CIS") {
                failingCount
                passingCount
                unknownCount
            }
        }
        count: nodeCount(query: $query)
    }
`;

export const defaultNodeSort = [
    {
        id: nodeSortFields.NODE,
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
            Header: `Node`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'name',
            id: nodeSortFields.NODE,
            sortField: nodeSortFields.NODE,
        },
        {
            Header: `Operating System`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'osImage',
            id: nodeSortFields.OPERATING_SYSTEM,
            sortField: nodeSortFields.OPERATING_SYSTEM,
        },
        {
            Header: `Container Runtime`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'containerRuntimeVersion',
            id: nodeSortFields.CONTAINER_RUNTIME,
            sortField: nodeSortFields.CONTAINER_RUNTIME,
        },
        {
            Header: `Node Join Time`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { joinedAt } = original;
                if (!joinedAt) {
                    return null;
                }
                return format(joinedAt, dateTimeFormat);
            },
            accessor: 'joinedAt',
            id: nodeSortFields.NODE_JOIN_TIME,
            sortField: nodeSortFields.NODE_JOIN_TIME,
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
                  id: nodeSortFields.CLUSTER,
                  sortField: nodeSortFields.CLUSTER,
              },
        entityContext && entityContext[entityTypes.CONTROL]
            ? null
            : {
                  Header: `CIS Controls`,
                  headerClassName: `w-1/8 ${nonSortableHeaderClassName}`,
                  className: `w-1/8 ${defaultColumnClassName}`,
                  accessor: 'nodeComplianceControlCount',
                  Cell: ({ original, pdf }) => {
                      const { nodeComplianceControlCount } = original;
                      const { passingCount, failingCount, unknownCount } =
                          nodeComplianceControlCount;
                      const controlCount = passingCount + failingCount + unknownCount;
                      if (!controlCount) {
                          return <NoEntitiesIconText text="No Controls" isTextOnly={pdf} />;
                      }
                      const url = URLService.getURL(match, location)
                          .push(original.id)
                          .push(entityTypes.CONTROL)
                          .url();
                      const text = `${controlCount} ${pluralize('Controls', controlCount)}`;
                      return (
                          <TableCellLink pdf={pdf} url={url}>
                              {text}
                          </TableCellLink>
                      );
                  },
                  sortable: false,
              },
    ];
    return tableColumns.filter((col) => col);
};

const createTableRows = (data) => data.results;

const Nodes = ({
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
            query={QUERY}
            variables={variables}
            entityType={entityTypes.NODE}
            tableColumns={tableColumns}
            createTableRows={createTableRows}
            onRowClick={onRowClick}
            selectedRowId={selectedRowId}
            idAttribute="id"
            defaultSorted={defaultNodeSort}
            data={data}
            totalResults={totalResults}
            autoFocusSearchInput={autoFocusSearchInput}
        />
    );
};
Nodes.propTypes = entityListPropTypes;
Nodes.defaultProps = entityListDefaultprops;

export default withRouter(Nodes);
