import React, { useContext } from 'react';
import pluralize from 'pluralize';
import gql from 'graphql-tag';

import queryService from 'modules/queryService';
import DateTimeField from 'Components/DateTimeField';
import LabelChip from 'Components/LabelChip';
import CVEStackedPill from 'Components/CVEStackedPill';
import TableCellLink from 'Components/TableCellLink';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import entityTypes from 'constants/entityTypes';
import workflowStateContext from 'Containers/workflowStateContext';
import { generateURLToFromTable } from 'modules/URLReadWrite';
import { DEPLOYMENT_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';

export const defaultDeploymentSort = [
    {
        id: 'priority',
        desc: false
    }
];

const VulnMgmtDeployments = ({ selectedRowId, search, entityContext, sort, page }) => {
    const workflowState = useContext(workflowStateContext);

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

    function getTableColumns() {
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
                    const url = generateURLToFromTable(workflowState, id, entityTypes.CVE);
                    return <CVEStackedPill vulnCounter={vulnCounter} url={url} pdf={pdf} />;
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
                          const url = generateURLToFromTable(workflowState, id, entityTypes.POLICY);
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
                    return policyStatus === 'pass' ? (
                        <LabelChip text="Pass" type="success" />
                    ) : (
                        <LabelChip text="Fail" type="alert" />
                    );
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
                          const url = generateURLToFromTable(
                              workflowState,
                              id,
                              entityTypes.CLUSTER,
                              clusterId
                          );
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
                          const url = generateURLToFromTable(
                              workflowState,
                              id,
                              entityTypes.NAMESPACE,
                              namespaceId
                          );
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
                    const url = generateURLToFromTable(workflowState, id, entityTypes.IMAGE);
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

    return (
        <WorkflowListPage
            query={query}
            queryOptions={queryOptions}
            entityListType={entityTypes.DEPLOYMENT}
            getTableColumns={getTableColumns}
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
