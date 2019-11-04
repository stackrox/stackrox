import React from 'react';
import pluralize from 'pluralize';
import gql from 'graphql-tag';

import queryService from 'modules/queryService';
import DateTimeField from 'Components/DateTimeField';
import StatusChip from 'Components/StatusChip';
import CVEStackedPill from 'Components/CVEStackedPill';
import TableCellLink from 'Components/TableCellLink';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import entityTypes from 'constants/entityTypes';
import { DEPLOYMENT_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';

export const defaultDeploymentSort = [
    {
        id: 'priority',
        desc: false
    }
];

export function getDeploymentTableColumns(workflowState) {
    const entityContext = workflowState.getBaseEntity();

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
        {
            Header: `CVEs`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { vulnCounter, id } = original;
                if (!vulnCounter || vulnCounter.all.total === 0) return 'No CVEs';

                const newState = workflowState.pushListItem(id).pushList(entityTypes.CVE);
                const url = newState.toUrl();
                const fixableUrl = newState.setSearch({ 'Is Fixable': true }).toUrl();

                return (
                    <CVEStackedPill
                        vulnCounter={vulnCounter}
                        url={url}
                        fixableUrl={fixableUrl}
                        hideLink={pdf}
                    />
                );
            }
        },
        {
            Header: `Latest violation`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { latestViolation } = original;
                return <DateTimeField date={latestViolation} />;
            },
            accessor: 'latestViolation'
        },
        entityContext[entityTypes.POLICY]
            ? null
            : {
                  Header: `Policies`,
                  headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                  className: `w-1/10 ${defaultColumnClassName}`,
                  accessor: 'failingPolicyCount',
                  Cell: ({ original, pdf }) => {
                      const { failingPolicyCount, id } = original;
                      if (failingPolicyCount === 0) return 'No failing policies';
                      const url = workflowState
                          .pushListItem(id)
                          .pushList(entityTypes.POLICY)
                          .toUrl();
                      return (
                          <TableCellLink
                              pdf={pdf}
                              url={url}
                              text={`${failingPolicyCount} ${pluralize(
                                  'policies',
                                  failingPolicyCount
                              )}`}
                          />
                      );
                  }
              },
        {
            Header: `Policy Status`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { policyStatus } = original;
                const policyLabel = <StatusChip status={policyStatus} />;

                return policyLabel;
            },
            id: 'policyStatus',
            accessor: 'policyStatus'
        },
        entityContext[entityTypes.CLUSTER]
            ? null
            : {
                  Header: `Cluster`,
                  headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                  className: `w-1/10 ${defaultColumnClassName}`,
                  accessor: 'clusterName',
                  Cell: ({ original, pdf }) => {
                      const { clusterName, clusterId, id } = original;
                      const url = workflowState
                          .pushListItem(id)
                          .pushRelatedEntity(entityTypes.CLUSTER, clusterId)
                          .toUrl();
                      return <TableCellLink pdf={pdf} url={url} text={clusterName} />;
                  }
              },
        entityContext[entityTypes.NAMESPACE]
            ? null
            : {
                  Header: `Namespace`,
                  headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                  className: `w-1/8 ${defaultColumnClassName}`,
                  accessor: 'namespace',
                  Cell: ({ original, pdf }) => {
                      const { namespace, namespaceId, id } = original;
                      const url = workflowState
                          .pushListItem(id)
                          .pushRelatedEntity(entityTypes.NAMESPACE, namespaceId)
                          .toUrl();
                      return <TableCellLink pdf={pdf} url={url} text={namespace} />;
                  }
              },
        {
            Header: `Images`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { imageCount, id } = original;
                if (imageCount === 0) return 'No images';
                const url = workflowState
                    .pushListItem(id)
                    .pushList(entityTypes.IMAGE)
                    .toUrl();
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
            Header: `Risk`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            accessor: 'priority'
        }
    ];
    return tableColumns.filter(col => col);
}

const VulnMgmtDeployments = ({ selectedRowId, search, sort, page }) => {
    const query = gql`
        query getDeployments($query: String) {
            results: deployments(query: $query) {
                ...deploymentListFields
            }
        }
        ${DEPLOYMENT_LIST_FRAGMENT}
    `;

    const queryOptions = {
        variables: {
            query: queryService.objectToWhereClause(search)
            // todo: add sort and page criteria
        }
    };

    return (
        <WorkflowListPage
            query={query}
            queryOptions={queryOptions}
            entityListType={entityTypes.DEPLOYMENT}
            getTableColumns={getDeploymentTableColumns}
            defaultSorted={defaultDeploymentSort}
            selectedRowId={selectedRowId}
            search={search}
            sort={sort}
            page={page}
        />
    );
};

VulnMgmtDeployments.propTypes = workflowListPropTypes;
VulnMgmtDeployments.defaultProps = workflowListDefaultProps;

export default VulnMgmtDeployments;
