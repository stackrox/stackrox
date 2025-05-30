import React from 'react';
import entityTypes from 'constants/entityTypes';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import { gql } from '@apollo/client';
import queryService from 'utils/queryService';
import { sortSeverity } from 'sorters/sorters';
import { BasePolicy } from 'types/policy.proto';

import NoResultsMessage from 'Components/NoResultsMessage';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PolicySeverityIconText from 'Components/PatternFly/IconText/PolicySeverityIconText';
import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';
import { formatLifecycleStages } from 'Containers/Policies/policies.utils';
import { getDateTime } from 'utils/dateUtils';
import TableWidget from './TableWidget';

type FailedPolicy = Pick<
    BasePolicy,
    'id' | 'name' | 'severity' | 'enforcementActions' | 'categories' | 'lifecycleStages'
>;

const QUERY = gql`
    query failedPolicies($query: String) {
        violations(query: $query) {
            id
            policy {
                id
                name
                severity
                enforcementActions
                categories
                lifecycleStages
            }
            time
        }
    }
`;

const createTableRows = (data: {
    violations: {
        id: string;
        time: string;
        policy: FailedPolicy;
    }[];
}) => {
    const initial: ({
        time: string;
    } & FailedPolicy)[] = [];
    const failedPolicies = data.violations.reduce((acc, curr) => {
        const row = {
            time: curr.time,
            ...curr.policy,
        };
        return [...acc, row];
    }, initial);
    return failedPolicies;
};

export type FailedPoliciesAcrossDeploymentProps = {
    deploymentID: string;
};

function FailedPoliciesAcrossDeployment({ deploymentID }: FailedPoliciesAcrossDeploymentProps) {
    if (!deploymentID) {
        return (
            <TableErrorComponent
                error={new Error('Unable to show failed policies for this deployment.')}
                message="A required ID for this deployment was not provided!"
            ></TableErrorComponent>
        );
    }

    return (
        <Query
            query={QUERY}
            variables={{
                query: queryService.objectToWhereClause({
                    'Deployment ID': deploymentID,
                    'Lifecycle Stage': 'DEPLOY',
                }),
            }}
        >
            {({ loading, data }) => {
                if (loading) {
                    return <Loader />;
                }
                if (!data) {
                    return null;
                }
                const rows = createTableRows(data);
                if (rows.length === 0) {
                    return (
                        <NoResultsMessage
                            message="No policies failed across this deployment"
                            className="p-3 shadow"
                            icon="info"
                        />
                    );
                }
                const header = `${rows.length} policies failed across this deployment`;
                const columns = [
                    {
                        Header: 'Id',
                        headerClassName: 'hidden',
                        className: 'hidden',
                        accessor: 'id',
                    },
                    {
                        Header: `Policy`,
                        headerClassName: `w-1/5 ${defaultHeaderClassName}`,
                        className: `w-1/5 ${defaultColumnClassName}`,
                        accessor: 'name',
                    },
                    {
                        Header: `Enforcing`,
                        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                        className: `w-1/8 ${defaultColumnClassName}`,
                        Cell: ({ original }) => {
                            const { enforcementActions } = original;
                            return (enforcementActions ?? []).length > 0 ? 'Yes' : 'No';
                        },
                        accessor: 'enforcementActions',
                    },
                    {
                        Header: `Severity`,
                        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                        className: `w-1/8 ${defaultColumnClassName}`,
                        Cell: ({ original, pdf }) => {
                            const { severity } = original;
                            return <PolicySeverityIconText severity={severity} isTextOnly={pdf} />;
                        },
                        accessor: 'severity',
                        sortMethod: sortSeverity,
                    },
                    {
                        Header: `Categories`,
                        headerClassName: `w-1/5 ${defaultHeaderClassName}`,
                        className: `w-1/5 ${defaultColumnClassName}`,
                        Cell: ({ original }) => {
                            const { categories }: { categories: string[] } = original;
                            return categories.join(', ');
                        },
                        accessor: 'categories',
                    },
                    {
                        Header: `Lifecycle Stage`,
                        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                        className: `w-1/8 ${defaultColumnClassName}`,
                        Cell: ({ original }) => {
                            const { lifecycleStages } = original;
                            return formatLifecycleStages(lifecycleStages);
                        },
                        accessor: 'lifecycleStages',
                    },
                    {
                        Header: 'Violation Time',
                        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                        className: `w-1/8 ${defaultColumnClassName}`,
                        Cell: ({ original }) => getDateTime(original.time),
                        accessor: 'time',
                    },
                ];
                return (
                    <TableWidget
                        entityType={entityTypes.POLICY}
                        header={header}
                        rows={rows}
                        columns={columns}
                        className="bg-base-100 w-full"
                        idAttribute="id"
                        noDataText="No failed policies."
                    />
                );
            }}
        </Query>
    );
}

export default FailedPoliciesAcrossDeployment;
