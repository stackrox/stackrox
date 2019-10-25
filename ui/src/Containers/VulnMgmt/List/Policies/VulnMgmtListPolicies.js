import React from 'react';
import pluralize from 'pluralize';
import gql from 'graphql-tag';

import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import DateTimeField from 'Components/DateTimeField';
import StatusChip from 'Components/StatusChip';
import SeverityLabel from 'Components/SeverityLabel';
import TableCellLink from 'Components/TableCellLink';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';
import entityTypes from 'constants/entityTypes';
import entityLabels from 'messages/entity';
import queryService from 'modules/queryService';
import { generateURLToFromTable } from 'modules/URLReadWrite';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';
import { sortDate } from 'sorters/sorters';

import { POLICY_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';

export function getCveTableColumns(workflowState) {
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
            headerClassName: `w-1/6 text-center ${defaultHeaderClassName}`,
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
            accessor: 'lastUpdated',
            sortMethod: sortDate
        },
        {
            Header: `Latest violation`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { latestViolation } = original;
                return <DateTimeField date={latestViolation} />;
            },
            accessor: 'latestViolation',
            sortMethod: sortDate
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
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { deploymentCount, id } = original;
                if (deploymentCount === 0) return 'No deployments';
                const url = generateURLToFromTable(workflowState, id, entityTypes.DEPLOYMENT);
                return (
                    <TableCellLink
                        pdf={pdf}
                        url={url}
                        text={`${deploymentCount} ${pluralize(
                            entityLabels.DEPLOYMENT,
                            deploymentCount
                        )}`}
                    />
                );
            },
            accessor: 'deploymentCount',
            id: 'deploymentCount'
        },
        {
            Header: `Lifecyle`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
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
            Cell: ({ original, pdf }) => {
                const { enforcementActions } = original;
                return enforcementActions || enforcementActions.length ? 'Yes' : 'No';
            },
            accessor: 'enforcementActions',
            id: 'enforcementActions'
        }
    ];

    return tableColumns.filter(col => col); // filter out columns that are nulled based on context
}

export const defaultPolicySort = [
    {
        id: 'name',
        desc: false
    }
];

const VulnMgmtPolicies = ({ selectedRowId, search, sort, page }) => {
    // TODO: change query line to `query getCves($query: String) {`
    //   after API starts accepting empty string ('') for query
    const POLICIES_QUERY = gql`
        query getPolicies {
            results: policies {
                ...policyListFields
            }
        }
        ${POLICY_LIST_FRAGMENT}
    `;

    const queryOptions = {
        variables: {
            query: queryService.objectToWhereClause(search)
        }
    };

    return (
        <WorkflowListPage
            query={POLICIES_QUERY}
            queryOptions={queryOptions}
            idAttribute="cve"
            entityListType={entityTypes.POLICY}
            getTableColumns={getCveTableColumns}
            selectedRowId={selectedRowId}
            search={search}
            sort={sort}
            page={page}
            defaultSorted={defaultPolicySort}
        />
    );
};

VulnMgmtPolicies.propTypes = workflowListPropTypes;
VulnMgmtPolicies.defaultProps = workflowListDefaultProps;

export default VulnMgmtPolicies;
