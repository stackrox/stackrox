import React from 'react';
import gql from 'graphql-tag';

import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import DateTimeField from 'Components/DateTimeField';
import StatusChip from 'Components/StatusChip';
import SeverityLabel from 'Components/SeverityLabel';
import TableCountLink from 'Components/workflow/TableCountLink';

import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';
import entityTypes from 'constants/entityTypes';
import queryService from 'modules/queryService';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';
import removeEntityContextColumns from 'utils/tableUtils';

import { POLICY_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';

export const defaultPolicySort = [
    {
        id: 'latestViolation',
        desc: true
    },
    {
        id: 'policyStatus',
        desc: false
    }
];

export function getPolicyTableColumns(workflowState) {
    // to determine whether to show the counts as links in the table when not in pure POLICY state
    const inFindingsSection = workflowState.getCurrentEntity().entityType !== entityTypes.POLICY;
    const tableColumns = [
        {
            Header: 'id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'id'
        },
        {
            Header: `Policy`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'name'
        },
        {
            Header: `Description`,
            headerClassName: `w-1/6 ${defaultHeaderClassName}`,
            className: `w-1/6 ${defaultColumnClassName}`,
            accessor: 'description',
            id: 'description'
        },
        {
            Header: `Policy status`,
            headerClassName: `w-1/10 text-center ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original }) => {
                const { policyStatus } = original;
                const policyLabel = <StatusChip status={policyStatus} />;

                return <div className="flex justify-center w-full">{policyLabel}</div>;
            },
            id: 'policyStatus',
            accessor: 'policyStatus'
        },
        {
            Header: `Last updated`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { lastUpdated } = original;
                return <DateTimeField date={lastUpdated} />;
            },
            accessor: 'lastUpdated'
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
            Header: `Severity`,
            headerClassName: `w-1/10 text-left ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { severity } = original;
                return <SeverityLabel severity={severity} />;
            },
            accessor: 'severity',
            id: 'severity'
        },
        {
            Header: `Deployments`,
            entityType: entityTypes.DEPLOYMENT,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => (
                <TableCountLink
                    entityType={entityTypes.DEPLOYMENT}
                    count={original.deploymentCount}
                    textOnly={inFindingsSection || pdf}
                    selectedRowId={original.id}
                />
            ),
            accessor: 'deploymentCount',
            id: 'deploymentCount'
        },
        {
            Header: `Lifecyle`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original }) => {
                const { lifecycleStages } = original;
                if (!lifecycleStages || !lifecycleStages.length) return 'No lifecycle stages';
                const lowercasedLifecycles = lifecycleStages
                    .map(stage => stage.toLowerCase())
                    .join(', ');

                return <span>{lowercasedLifecycles}</span>;
            },
            accessor: 'lifecycleStages',
            id: 'lifecycleStages'
        },
        {
            Header: `Enforcement`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original }) => {
                const { enforcementActions } = original;
                return enforcementActions || enforcementActions.length ? 'Yes' : 'No';
            },
            accessor: 'enforcementActions',
            id: 'enforcementActions'
        }
    ];

    return removeEntityContextColumns(tableColumns, workflowState);
}

const VulnMgmtPolicies = ({ selectedRowId, search, sort, page, data }) => {
    const POLICIES_QUERY = gql`
        query getPolicies($policyQuery: String) {
            results: policies(query: $policyQuery) {
                ...policyFields
            }
        }
        ${POLICY_LIST_FRAGMENT}
    `;

    const queryOptions = {
        variables: {
            policyQuery: queryService.objectToWhereClause({
                ...search,
                Category: 'Vulnerability Management'
            })
        }
    };

    return (
        <WorkflowListPage
            data={data}
            query={POLICIES_QUERY}
            queryOptions={queryOptions}
            idAttribute="id"
            entityListType={entityTypes.POLICY}
            getTableColumns={getPolicyTableColumns}
            selectedRowId={selectedRowId}
            search={search}
            page={page}
            defaultSorted={sort || defaultPolicySort}
        />
    );
};

VulnMgmtPolicies.propTypes = workflowListPropTypes;
VulnMgmtPolicies.defaultProps = {
    ...workflowListDefaultProps,
    sort: null
};

export default VulnMgmtPolicies;
