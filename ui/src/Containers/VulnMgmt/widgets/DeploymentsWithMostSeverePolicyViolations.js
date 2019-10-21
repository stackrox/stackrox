import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { Link } from 'react-router-dom';
import { useQuery } from 'react-apollo';
import gql from 'graphql-tag';
import { severityValues, severities } from 'constants/severities';
import queryService from 'modules/queryService';
import sortBy from 'lodash/sortBy';

import WorkflowStateMgr from 'modules/WorkflowStateManager';
import workflowStateContext from 'Containers/workflowStateContext';
import { generateURL } from 'modules/URLReadWrite';

import Button from 'Components/Button';
import Loader from 'Components/Loader';
import Widget from 'Components/Widget';
import NumberedList from 'Components/NumberedList';
import LabelChip from 'Components/LabelChip';

const DEPLOYMENTS_WITH_MOST_SEVERE_POLICY_VIOLATIONS = gql`
    query deploymentsWithMostSeverePolicyViolations($query: String, $pagination: Pagination) {
        results: deployments(query: $query, pagination: $pagination) {
            id
            name
            failingPolicies {
                id
                severity
            }
        }
    }
`;

const ViewAllButton = ({ url }) => {
    return (
        <Link to={url} className="no-underline">
            <Button className="btn-sm btn-base" type="button" text="View All" />
        </Link>
    );
};

const getViewAllURL = workflowState => {
    const workflowStateMgr = new WorkflowStateMgr(workflowState);
    workflowStateMgr.pushList(entityTypes.DEPLOYMENT);
    const url = generateURL(workflowStateMgr.workflowState);
    return url;
};

const getSingleEntityURL = (workflowState, id) => {
    const workflowStateMgr = new WorkflowStateMgr(workflowState);
    workflowStateMgr.pushList(entityTypes.DEPLOYMENT).pushListItem(id);
    const url = generateURL(workflowStateMgr.workflowState);
    return url;
};

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

const processData = (data, workflowState) => {
    const results = sortBy(data.results, [sortBySevereViolations])
        .splice(-8)
        .reverse(); // @TODO: Remove when we have pagination on Policies
    return results.map(({ id, name, failingPolicies }) => {
        const text = name;
        const { critical, high, medium, low } = getPolicySeverityCounts(failingPolicies);
        return {
            text,
            url: getSingleEntityURL(workflowState, id),
            component: (
                <>
                    <div className="mr-4">
                        <LabelChip text={`${low} L`} type="base" />
                    </div>
                    <div className="mr-4">
                        <LabelChip text={`${medium} M`} type="warning" />
                    </div>
                    <div className="mr-4">
                        <LabelChip text={`${high} H`} type="caution" />
                    </div>
                    <LabelChip text={`${critical} C`} type="alert" />
                </>
            )
        };
    });
};

const DeploymentsWithMostSeverePolicyViolations = ({ entityContext }) => {
    const { loading, data = {} } = useQuery(DEPLOYMENTS_WITH_MOST_SEVERE_POLICY_VIOLATIONS, {
        variables: {
            query: queryService.entityContextToQueryString(entityContext),
            pagination: {
                limit: 8,
                sortOption: {
                    field: 'priority',
                    reversed: false
                }
            }
        }
    });

    let content = <Loader />;

    const workflowState = useContext(workflowStateContext);
    if (!loading) {
        const processedData = processData(data, workflowState);

        content = (
            <div className="w-full">
                <NumberedList data={processedData} />
            </div>
        );
    }

    return (
        <Widget
            className="s-2 pdf-page"
            header="Deployments With Most Severe Policy Violations"
            headerComponents={<ViewAllButton url={getViewAllURL(workflowState)} />}
        >
            {content}
        </Widget>
    );
};

DeploymentsWithMostSeverePolicyViolations.propTypes = {
    entityContext: PropTypes.shape({})
};

DeploymentsWithMostSeverePolicyViolations.defaultProps = {
    entityContext: {}
};

export default DeploymentsWithMostSeverePolicyViolations;
