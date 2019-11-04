import React, { useContext } from 'react';
import pluralize from 'pluralize';
import gql from 'graphql-tag';

import queryService from 'modules/queryService';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import DateTimeField from 'Components/DateTimeField';
import LabelChip from 'Components/LabelChip';
import StatusChip from 'Components/StatusChip';
import TableCellLink from 'Components/TableCellLink';
import entityTypes from 'constants/entityTypes';
import workflowStateContext from 'Containers/workflowStateContext';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';
import CVEStackedPill from 'Components/CVEStackedPill';

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
            {
                Header: `CVEs`,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => {
                    const { vulnCounter, id } = original;
                    if (!vulnCounter || vulnCounter.all.total === 0) return 'No CVEs';
                    const url = workflowState
                        .pushListItem(id)
                        .pushList(entityTypes.CVE)
                        .toUrl();

                    return <CVEStackedPill vulnCounter={vulnCounter} url={url} pdf={pdf} />;
                }
            },
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
                headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                // eslint-disable-next-line
                Cell: ({ original, pdf }) => {
                    const { namespaceCount } = original;
                    if (!namespaceCount) {
                        return <LabelChip text="No Namespaces" type="alert" />;
                    }
                    const url = workflowState
                        .pushListItem(original.id)
                        .pushList(entityTypes.NAMESPACE)
                        .toUrl();

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
                headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                // eslint-disable-next-line
                Cell: ({ original, pdf }) => {
                    const { deploymentCount } = original;
                    if (!deploymentCount) {
                        return <LabelChip text="No Deployments" type="alert" />;
                    }
                    const url = workflowState
                        .pushListItem(original.id)
                        .pushList(entityTypes.DEPLOYMENT)
                        .toUrl();

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
                headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                // eslint-disable-next-line
                Cell: ({ original, pdf }) => {
                    const { policyCount } = original;
                    if (!policyCount) {
                        return <LabelChip text="No Policies" type="alert" />;
                    }
                    const url = workflowState
                        .pushListItem(original.id)
                        .pushList(entityTypes.POLICY)
                        .toUrl();

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
                headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                // eslint-disable-next-line
                Cell: ({ original }) => {
                    const { policyStatus } = original;
                    const policyLabel = <StatusChip status={policyStatus && policyStatus.status} />;

                    return policyLabel;
                },
                id: 'policyStatus'
            },
            {
                Header: `Latest violation`,
                headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                Cell: ({ original }) => {
                    const { latestViolation } = original;
                    return <DateTimeField date={latestViolation} />;
                },
                accessor: 'latestViolation'
            },
            {
                Header: `Risk Priority`,
                headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                accessor: 'priority'
            }
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
