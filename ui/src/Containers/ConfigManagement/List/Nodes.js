import React from 'react';
import gql from 'graphql-tag';
import entityTypes from 'constants/entityTypes';
import URLService from 'modules/URLService';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import queryService from 'modules/queryService';
import { format } from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';
import { sortDate } from 'sorters/sorters';

import LabelChip from 'Components/LabelChip';
import pluralize from 'pluralize';
import { withRouter } from 'react-router-dom';
import List from './List';
import TableCellLink from './Link';

const QUERY = gql`
    query nodes($query: String) {
        results: nodes(query: $query) {
            id
            name
            clusterName
            clusterId
            osImage
            containerRuntimeVersion
            joinedAt
            complianceResults(query: "Standard: CIS") {
                resource {
                    __typename
                }
                control {
                    id
                }
            }
        }
    }
`;

const buildTableColumns = (match, location, entityContext) => {
    const tableColumns = [
        {
            Header: 'Id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'id'
        },
        {
            Header: `Node`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'name'
        },
        {
            Header: `Operating System`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'osImage'
        },
        {
            Header: `Container Runtime`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'containerRuntimeVersion'
        },
        {
            Header: `Node join time`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { joinedAt } = original;
                if (!joinedAt) return null;
                return format(joinedAt, dateTimeFormat);
            },
            accessor: 'joinedAt',
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
            Header: `CIS Controls`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'complianceResults',
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { complianceResults = [] } = original;
                const filteredComplianceResults = complianceResults.filter(
                    // eslint-disable-next-line
                    result => result.resource.__typename === 'Node'
                );
                const { length } = filteredComplianceResults;
                if (!length) {
                    return <LabelChip text="No Controls" type="alert" />;
                }
                const url = URLService.getURL(match, location)
                    .push(original.id)
                    .push(entityTypes.CONTROL)
                    .url();
                return (
                    <TableCellLink
                        pdf={pdf}
                        url={url}
                        text={`${length} ${pluralize('Controls', length)}`}
                    />
                );
            }
        }
    ];
    return tableColumns.filter(col => col);
};

const createTableRows = data => data.results;

const Nodes = ({
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
            entityType={entityTypes.NODE}
            tableColumns={tableColumns}
            createTableRows={createTableRows}
            onRowClick={onRowClick}
            selectedRowId={selectedRowId}
            idAttribute="id"
            data={data}
        />
    );
};
Nodes.propTypes = entityListPropTypes;
Nodes.defaultProps = entityListDefaultprops;

export default withRouter(Nodes);
