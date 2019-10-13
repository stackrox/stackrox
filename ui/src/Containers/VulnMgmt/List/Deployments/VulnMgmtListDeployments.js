import React, { useContext } from 'react';
import pluralize from 'pluralize';
import gql from 'graphql-tag';

import queryService from 'modules/queryService';
import LabelChip from 'Components/LabelChip';
import TableCellLink from 'Components/TableCellLink';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import entityTypes from 'constants/entityTypes';
import WorkflowStateMgr from 'modules/WorkflowStateManager';
import workflowStateContext from 'Containers/workflowStateContext';
import { generateURL } from 'modules/URLReadWrite';
import { DEPLOYMENT_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import { PropTypes } from 'prop-types';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';

const VulnMgmtDeployments = ({ selectedRowId, search, entityContext }) => {
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
            entityContext[entityTypes.CLUSTER]
                ? null
                : {
                      Header: `Cluster`,
                      headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                      className: `w-1/8 ${defaultColumnClassName}`,
                      accessor: 'clusterName',
                      Cell: ({ original, pdf }) => {
                          const { clusterName, clusterId, id } = original;
                          const workflowStateMgr = new WorkflowStateMgr(workflowState);
                          workflowStateMgr
                              .pushListItem(id)
                              .pushRelatedEntity(entityTypes.CLUSTER, clusterId);
                          const url = generateURL(workflowStateMgr.workflowState);
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
                          const workflowStateMgr = new WorkflowStateMgr(workflowState);
                          workflowStateMgr
                              .pushListItem(id)
                              .pushRelatedEntity(entityTypes.NAMESPACE, namespaceId);
                          const url = generateURL(workflowStateMgr.workflowState);
                          return <TableCellLink pdf={pdf} url={url} text={namespace} />;
                      }
                  },
            {
                Header: `Policy Status`,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                Cell: ({ original }) => {
                    const { policyStatus } = original;
                    return policyStatus === 'pass' ? (
                        'Pass'
                    ) : (
                        <LabelChip text="Fail" type="alert" />
                    );
                },
                id: 'policyStatus',
                accessor: 'policyStatus'
            },
            {
                Header: `Images`,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => {
                    const { imageCount, id } = original;
                    if (imageCount === 0) return 'No images';
                    const workflowStateMgr = new WorkflowStateMgr(workflowState);
                    workflowStateMgr.pushListItem(id).pushList(entityTypes.IMAGE);
                    const url = generateURL(workflowStateMgr.workflowState);
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
                Cell: ({ original, pdf }) => {
                    const { secretCount, id } = original;
                    if (secretCount === 0) return 'No secrets';
                    const workflowStateMgr = new WorkflowStateMgr(workflowState);
                    workflowStateMgr.pushListItem(id).pushList(entityTypes.SECRET);
                    const url = generateURL(workflowStateMgr.workflowState);
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
            entityContext[entityTypes.SERVICE_ACCOUNT]
                ? null
                : {
                      Header: `Service Account`,
                      headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                      className: `w-1/8 ${defaultColumnClassName}`,
                      accessor: 'serviceAccount',
                      Cell: ({ original, pdf }) => {
                          const { serviceAccount, serviceAccountID, id } = original;
                          const workflowStateMgr = new WorkflowStateMgr(workflowState);
                          workflowStateMgr
                              .pushListItem(id)
                              .pushRelatedEntity(entityTypes.SERVICE_ACCOUNT, serviceAccountID);
                          const url = generateURL(workflowStateMgr.workflowState);
                          return <TableCellLink pdf={pdf} url={url} text={serviceAccount} />;
                      }
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
            selectedRowId={selectedRowId}
            search={search}
        />
    );
};

VulnMgmtDeployments.propTypes = {
    selectedRowId: PropTypes.string,
    search: PropTypes.shape({}),
    entityContext: PropTypes.shape({})
};

VulnMgmtDeployments.defaultProps = {
    search: null,
    entityContext: {},
    selectedRowId: null
};

export default VulnMgmtDeployments;
