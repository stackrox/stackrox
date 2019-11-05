import React from 'react';
import gql from 'graphql-tag';
import pluralize from 'pluralize';

import queryService from 'modules/queryService';
import TableCellLink from 'Components/TableCellLink';
import CVEStackedPill from 'Components/CVEStackedPill';
import StatusChip from 'Components/StatusChip';
import DateTimeField from 'Components/DateTimeField';
import { sortDate } from 'sorters/sorters';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import entityTypes from 'constants/entityTypes';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';
import { NAMESPACE_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';

export function getNamespaceTableColumns(workflowState) {
    const tableColumns = [
        {
            Header: 'Id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'metadata.id'
        },
        {
            Header: `Namespace`,
            headerClassName: `w-1/6 ${defaultHeaderClassName}`,
            className: `w-1/6 ${defaultColumnClassName}`,
            accessor: 'metadata.name'
        },
        {
            Header: `CVEs`,
            headerClassName: `w-1/6 ${defaultHeaderClassName}`,
            className: `w-1/6 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { vulnCounter, metadata } = original;
                if (!vulnCounter || vulnCounter.all.total === 0) return 'No CVEs';

                const newState = workflowState.pushListItem(metadata.id).pushList(entityTypes.CVE);
                const cvesUrl = newState.toUrl();
                const fixableUrl = newState.setSearch({ 'Is Fixable': true }).toUrl();

                return (
                    <CVEStackedPill
                        vulnCounter={vulnCounter}
                        url={cvesUrl}
                        fixableUrl={fixableUrl}
                        hideLink={pdf}
                    />
                );
            },
            accessor: 'vulnCounter.all.total'
        },
        {
            Header: `Cluster`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { metadata } = original;
                const { clusterName, clusterId, id } = metadata;
                const url = workflowState
                    .pushListItem(id)
                    .pushRelatedEntity(entityTypes.CLUSTER, clusterId)
                    .toUrl();

                return <TableCellLink pdf={pdf} url={url} text={clusterName} />;
            },
            accessor: 'metadata.clusterName'
        },
        {
            Header: `Deployments`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { deploymentCount, metadata } = original;
                const url = workflowState
                    .pushListItem(metadata.id)
                    .pushList(entityTypes.DEPLOYMENT)
                    .toUrl();

                const text = `${deploymentCount} ${pluralize(
                    entityTypes.DEPLOYMENT.toLowerCase(),
                    deploymentCount
                )}`;
                return <TableCellLink pdf={pdf} url={url} text={text} />;
            },
            accessor: 'deploymentCount'
        },
        {
            Header: `Images`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { imageCount, metadata } = original;
                const url = workflowState
                    .pushListItem(metadata.id)
                    .pushList(entityTypes.IMAGE)
                    .toUrl();

                const text = `${imageCount} ${pluralize(
                    entityTypes.IMAGE.toLowerCase(),
                    imageCount
                )}`;
                return <TableCellLink pdf={pdf} url={url} text={text} />;
            },
            accessor: 'imageCount'
        },
        {
            Header: `Policies`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { policyCount, metadata } = original;
                const url = workflowState
                    .pushListItem(metadata.id)
                    .pushList(entityTypes.POLICY)
                    .toUrl();
                const text = `${policyCount} ${pluralize(
                    entityTypes.POLICY.toLowerCase(),
                    policyCount
                )}`;
                return <TableCellLink pdf={pdf} url={url} text={text} />;
            },
            accessor: 'policyCount'
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
            id: 'policyStatus',
            accessor: 'policyStatus.status'
        },
        {
            Header: `Latest violation`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { latestViolation } = original;
                return <DateTimeField date={latestViolation} />;
            },
            accessor: 'latestViolation',
            sortMethod: sortDate
        },
        {
            Header: `Risk Priority`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            accessor: 'metadata.priority'
        }
    ];
    return tableColumns.filter(col => col);
}

const VulnMgmtNamespaces = ({ selectedRowId, search, sort, page }) => {
    const query = gql`
        query getNamespaces {
            results: namespaces {
                ...namespaceListFields
            }
        }
        ${NAMESPACE_LIST_FRAGMENT}
    `;

    const queryOptions = {
        variables: {
            query: queryService.objectToWhereClause(search)
        }
    };

    const defaultNamespaceSort = [
        {
            id: 'metadata.priority',
            desc: false
        }
    ];

    return (
        <WorkflowListPage
            query={query}
            queryOptions={queryOptions}
            entityListType={entityTypes.NAMESPACE}
            getTableColumns={getNamespaceTableColumns}
            selectedRowId={selectedRowId}
            idAttribute="metadata.id"
            search={search}
            sort={sort}
            page={page}
            defaultSorted={defaultNamespaceSort}
        />
    );
};

VulnMgmtNamespaces.propTypes = workflowListPropTypes;
VulnMgmtNamespaces.defaultProps = workflowListDefaultProps;

export default VulnMgmtNamespaces;
