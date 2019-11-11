import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { useQuery } from 'react-apollo';
import gql from 'graphql-tag';
import { severityValues, severities } from 'constants/severities';
import queryService from 'modules/queryService';
import sortBy from 'lodash/sortBy';

import workflowStateContext from 'Containers/workflowStateContext';

import ViewAllButton from 'Components/ViewAllButton';
import Loader from 'Components/Loader';
import Widget from 'Components/Widget';
import NumberedList from 'Components/NumberedList';
import LabelChip from 'Components/LabelChip';
import NoResultsMessage from 'Components/NoResultsMessage';

const DEPLOYMENTS_WITH_MOST_SEVERE_POLICY_VIOLATIONS = gql`
    query deploymentsWithMostSeverePolicyViolations(
        $query: String
        $policyQuery: String
        $pagination: Pagination
    ) {
        results: deployments(query: $query, pagination: $pagination) {
            id
            name
            failingPolicies(query: $policyQuery) {
                id
                severity
            }
        }
    }
`;

const getPolicySeverityCounts = failingPolicies => {
    const counts = failingPolicies.reduce(
        (acc, curr) => {
            acc[curr.severity] += 1;
            return acc;
        },
        {
            [severities.CRITICAL_SEVERITY]: 0,
            [severities.HIGH_SEVERITY]: 0,
            [severities.MEDIUM_SEVERITY]: 0,
            [severities.LOW_SEVERITY]: 0
        }
    );
    return {
        critical: counts.CRITICAL_SEVERITY,
        high: counts.HIGH_SEVERITY,
        medium: counts.MEDIUM_SEVERITY,
        low: counts.LOW_SEVERITY
    };
};

const sortBySevereViolations = datum => {
    return datum.failingPolicies.reduce((acc, curr) => {
        return acc + severityValues[curr.severity];
    }, 0);
};

const processData = (data, workflowState, limit) => {
    const results = sortBy(data.results, [sortBySevereViolations])
        .slice(-limit)
        .reverse(); // @TODO: Remove when we have pagination on Policies
    return results.map(({ id, name, failingPolicies }) => {
        const text = name;
        const { critical, high, medium, low } = getPolicySeverityCounts(failingPolicies);
        return {
            text,
            url: workflowState.pushRelatedEntity(entityTypes.DEPLOYMENT, id).toUrl(),
            component: (
                <>
                    <div className="mr-4">
                        <LabelChip text={`${low} L`} type="base" size="small" />
                    </div>
                    <div className="mr-4">
                        <LabelChip text={`${medium} M`} type="warning" size="small" />
                    </div>
                    <div className="mr-4">
                        <LabelChip text={`${high} H`} type="caution" size="small" />
                    </div>
                    <LabelChip text={`${critical} C`} type="alert" size="small" />
                </>
            )
        };
    });
};

const DeploymentsWithMostSeverePolicyViolations = ({ entityContext, limit }) => {
    const { loading, data = {} } = useQuery(DEPLOYMENTS_WITH_MOST_SEVERE_POLICY_VIOLATIONS, {
        variables: {
            query: queryService.entityContextToQueryString(entityContext),
            policyQuery: queryService.objectToWhereClause({
                Category: 'Vulnerability Management'
            })
        }
    });

    let content = <Loader />;

    const workflowState = useContext(workflowStateContext);
    const viewAllURL = workflowState
        .pushList(entityTypes.DEPLOYMENT)
        .setSort([
            { id: 'policyStatus', desc: false },
            { id: 'failingPolicyCount', desc: true },
            { id: 'name', desc: false }
        ])
        .toUrl();

    if (!loading) {
        const processedData = processData(data, workflowState, limit);

        if (!processedData || processedData.length === 0) {
            content = (
                <NoResultsMessage message="No deployments found" className="p-6" icon="info" />
            );
        } else {
            content = (
                <div className="w-full">
                    <NumberedList data={processedData} />
                </div>
            );
        }
    }

    return (
        <Widget
            className="h-full pdf-page"
            header="Deployments With Most Severe Policy Violations"
            headerComponents={<ViewAllButton url={viewAllURL} />}
        >
            {content}
        </Widget>
    );
};

DeploymentsWithMostSeverePolicyViolations.propTypes = {
    entityContext: PropTypes.shape({}),
    limit: PropTypes.number
};

DeploymentsWithMostSeverePolicyViolations.defaultProps = {
    entityContext: {},
    limit: 5
};

export default DeploymentsWithMostSeverePolicyViolations;
