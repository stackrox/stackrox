import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { withRouter } from 'react-router-dom';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import { gql } from '@apollo/client';
import queryService from 'utils/queryService';
import { sortSeverity } from 'sorters/sorters';
import { format } from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';

import NoResultsMessage from 'Components/NoResultsMessage';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PolicySeverityIconText from 'Components/PatternFly/IconText/PolicySeverityIconText';
import { formatLifecycleStages } from 'Containers/Policies/policies.utils';
import TableWidget from './TableWidget';

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

const createTableRows = (data) => {
    const failedPolicies = data.violations.reduce((acc, curr) => {
        const row = {
            time: curr.time,
            ...curr.policy,
        };
        return [...acc, row];
    }, []);
    return failedPolicies;
};

const FailedPoliciesAcrossDeployment = ({ deploymentID }) => {
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
                        Header: `Enforced`,
                        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                        className: `w-1/8 ${defaultColumnClassName}`,
                        Cell: ({ original }) => {
                            const { enforcementActions } = original;
                            return enforcementActions ? 'Yes' : 'No';
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
                            const { categories } = original;
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
                        Cell: ({ original }) => format(original.time, dateTimeFormat),
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
};

FailedPoliciesAcrossDeployment.propTypes = {
    deploymentID: PropTypes.string.isRequired,
};

export default withRouter(FailedPoliciesAcrossDeployment);
