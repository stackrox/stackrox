import React from 'react';
import PropTypes from 'prop-types';
import gql from 'graphql-tag';
import queryService from 'modules/queryService';
import { format } from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import Widget from 'Components/Widget';

const QUERY = gql`
    query violationsInDeployment($query: String) {
        violations(query: $query) {
            id
            time
            policy {
                id
                enforcementActions
                categories
            }
            violations {
                message
                link
            }
        }
    }
`;

const processData = data => {
    if (!data.violations || !data.violations.length) return null;
    return data.violations[0];
};

const ViolationsAcrossThisDeployment = ({ deploymentID, policyID }) => {
    const variables = {
        query: queryService.objectToWhereClause({
            'Deployment ID': deploymentID,
            'Policy ID': policyID
        })
    };
    return (
        <Query query={QUERY} variables={variables}>
            {({ loading, data }) => {
                if (loading) return <Loader />;
                if (!data) return null;
                const policyViolation = processData(data);
                let content = null;
                if (policyViolation) {
                    content = (
                        <>
                            <div>
                                <Widget
                                    header="Time of Violation"
                                    className="m-4"
                                    bodyClassName="flex flex-col p-4 leading-normal"
                                >
                                    {format(policyViolation.time, dateTimeFormat)}
                                </Widget>
                                <Widget
                                    header="Enforcement"
                                    className="m-4"
                                    bodyClassName="flex flex-col p-4 leading-normal"
                                >
                                    {policyViolation.policy.enforcementActions.join(', ') ||
                                        'No Enforcement'}
                                </Widget>
                            </div>
                            <Widget
                                header="Violation"
                                className="m-4"
                                bodyClassName="flex flex-col p-4 leading-normal"
                            >
                                <ul className="pl-3 leading-loose">
                                    {policyViolation.violations.map(violation => {
                                        return <li>{violation.message}</li>;
                                    })}
                                </ul>
                            </Widget>
                            <Widget
                                header="Category"
                                className="m-4"
                                bodyClassName="flex flex-col p-4 leading-normal"
                            >
                                {policyViolation.policy.categories.join(', ')}
                            </Widget>
                        </>
                    );
                } else {
                    content = <div className="p-4">No Violations</div>;
                }
                return <div className="flex w-full mx-4 shadow rounded bg-base-100">{content}</div>;
            }}
        </Query>
    );
};

ViolationsAcrossThisDeployment.propTypes = {
    deploymentID: PropTypes.string.isRequired,
    policyID: PropTypes.string.isRequired
};

export default ViolationsAcrossThisDeployment;
