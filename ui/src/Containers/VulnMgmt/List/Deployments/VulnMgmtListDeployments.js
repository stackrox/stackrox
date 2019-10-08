import React, { useContext } from 'react';
import pluralize from 'pluralize';

// TODO refactor out
import gql from 'graphql-tag';

import LabelChip from 'Components/LabelChip';
import TableCellLink from 'Components/TableCellLink';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import entityTypes from 'constants/entityTypes';
import WorkflowStateMgr from 'modules/WorkflowStateManager';
import { generateURL } from 'modules/URLReadWrite';
import workflowStateContext from 'Containers/workflowStateContext';
import VulnMgmtListTable from '../VulnMgmtListTable';

// TODO refactor this out to a common place
import filterByPolicyStatus from '../../../ConfigManagement/List/utilities/filterByPolicyStatus';

// TODO update with vulnerability-specific fields, after they become available
const DEPLOYMENTS_QUERY = gql`
    query getDeployments($query: String) {
        results: deployments(query: $query) {
            id
            name
            clusterName
            clusterId
            namespace
            namespaceId
            serviceAccount
            serviceAccountID
            secretCount
            imageCount
            policyStatus
        }
    }
`;

const buildTableColumns = (entityContext, workflowState) => {
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
                  // eslint-disable-next-line
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
                  // eslint-disable-next-line
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
            // eslint-disable-next-line
            Cell: ({ original }) => {
                const { policyStatus } = original;
                return policyStatus === 'pass' ? 'Pass' : <LabelChip text="Fail" type="alert" />;
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
            // eslint-disable-next-line
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
                  // eslint-disable-next-line
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
};

const VulnMgmtDeployments = ({
    wrapperClass,
    entityContext = {},
    loading,
    error,
    data,
    selectedRowId
}) => {
    const workflowState = useContext(workflowStateContext);
    // TODO: figure out where policyStatus comes from?
    const policyStatus = null;

    const tableColumns = buildTableColumns(entityContext, workflowState);

    function createTableRowsFilteredByPolicyStatus(items) {
        const tableRows = items.results || items || []; // guard to pluck data from different API returs

        const filteredTableRows = filterByPolicyStatus(tableRows, policyStatus);
        return filteredTableRows;
    }

    // TODO: refactor to remove the need for the intermediate <VulnMgmtListTable> component
    return (
        <VulnMgmtListTable
            wrapperClass={wrapperClass}
            query={DEPLOYMENTS_QUERY}
            entityType={entityTypes.DEPLOYMENT}
            tableColumns={tableColumns}
            createTableRows={createTableRowsFilteredByPolicyStatus}
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
            defaultSearchOptions={null}
            loading={loading}
            error={error}
            data={filterByPolicyStatus(data, policyStatus)}
        />
    );
};

export default VulnMgmtDeployments;
