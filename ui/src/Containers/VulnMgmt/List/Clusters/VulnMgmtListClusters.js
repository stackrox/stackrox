import React, { useContext } from 'react';
import pluralize from 'pluralize';
import gql from 'graphql-tag';

import queryService from 'modules/queryService';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import LabelChip from 'Components/LabelChip';
import TableCellLink from 'Components/TableCellLink';
import entityTypes from 'constants/entityTypes';
import workflowStateContext from 'Containers/workflowStateContext';
import WorkflowStateMgr from 'modules/WorkflowStateManager';
import { generateURL } from 'modules/URLReadWrite';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';

import { CLUSTER_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';

const VulnMgmtClusters = ({ selectedRowId, search, sort, page }) => {
    const workflowState = useContext(workflowStateContext);

    const query = gql`
        query getClusters($query: String) {
            results: clusters(query: $query) {
                ...clusterListFields
            }
        }
        ${CLUSTER_LIST_FRAGMENT}
    `;

    const queryOptions = {
        variables: search
            ? {
                  query: queryService.objectToWhereClause(search)
              }
            : null
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
                Header: `Cluster`,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                accessor: 'name'
            },
            // {
            // TODO: enable this column after data is available from the API
            //     Header: `CVEs`,
            //     headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            //     className: `w-1/8 ${defaultColumnClassName}`,
            //     accessor: 'cves'
            // },
            {
                Header: `K8S version`,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                accessor: 'status.orchestratorMetadata.version'
            },
            // TODO: enable this column after data is available from the API
            // {
            //     Header: `Created`,
            //     headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            //     className: `w-1/8 ${defaultColumnClassName}`,
            //     accessor: 'createdAt'
            // },
            {
                Header: `Namespaces`,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                // eslint-disable-next-line
                Cell: ({ original, pdf }) => {
                    const { namespaceCount } = original;
                    if (!namespaceCount) {
                        return <LabelChip text="No Namespaces" type="alert" />;
                    }
                    const workflowStateMgr = new WorkflowStateMgr(workflowState);
                    workflowStateMgr.pushListItem(original.id).pushList(entityTypes.NAMESPACE);
                    const url = generateURL(workflowStateMgr.workflowState);
                    return (
                        <TableCellLink
                            pdf={pdf}
                            url={url}
                            text={`${namespaceCount} ${pluralize('Namespace', namespaceCount)}`}
                        />
                    );
                }
            },
            {
                Header: `Deployments`,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                // eslint-disable-next-line
                Cell: ({ original, pdf }) => {
                    const { deploymentCount } = original;
                    if (!deploymentCount) {
                        return <LabelChip text="No Deployments" type="alert" />;
                    }
                    const workflowStateMgr = new WorkflowStateMgr(workflowState);
                    workflowStateMgr.pushListItem(original.id).pushList(entityTypes.DEPLOYMENT);
                    const url = generateURL(workflowStateMgr.workflowState);
                    return (
                        <TableCellLink
                            pdf={pdf}
                            url={url}
                            text={`${deploymentCount} ${pluralize('Deployment', deploymentCount)}`}
                        />
                    );
                },
                id: 'deploymentCount'
            },
            {
                Header: `Policies`,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                // eslint-disable-next-line
                Cell: ({ original, pdf }) => {
                    const { policyCount } = original;
                    if (!policyCount) {
                        return <LabelChip text="No Policies" type="alert" />;
                    }
                    const workflowStateMgr = new WorkflowStateMgr(workflowState);
                    workflowStateMgr.pushListItem(original.id).pushList(entityTypes.POLICY);
                    const url = generateURL(workflowStateMgr.workflowState);
                    return (
                        <TableCellLink
                            pdf={pdf}
                            url={url}
                            text={`${policyCount} ${pluralize('Policy', policyCount)}`}
                        />
                    );
                },
                id: 'policyCount'
            },
            {
                Header: `Policy status`,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                // eslint-disable-next-line
                Cell: ({ original }) => {
                    const { policyStatus } = original;
                    return policyStatus.status === 'pass' ? (
                        <LabelChip text="Pass" type="success" />
                    ) : (
                        <LabelChip text="Fail" type="alert" />
                    );
                },
                id: 'policyStatus'
            } // ,
            // {
            //     Header: `Latest violation`,
            //     headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            //     className: `w-1/8 ${defaultColumnClassName}`,
            //     accessor: 'latestViolation'
            // },
            // {
            //     Header: `Risk`,
            //     headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            //     className: `w-1/8 ${defaultColumnClassName}`,
            //     accessor: 'risk'
            // }
        ];
        return tableColumns;
    }

    return (
        <WorkflowListPage
            query={query}
            queryOptions={queryOptions}
            entityListType={entityTypes.CLUSTER}
            getTableColumns={getTableColumns}
            selectedRowId={selectedRowId}
            search={search}
            sort={sort}
            page={page}
        />
    );
};

VulnMgmtClusters.propTypes = workflowListPropTypes;
VulnMgmtClusters.defaultProps = workflowListDefaultProps;

export default VulnMgmtClusters;
