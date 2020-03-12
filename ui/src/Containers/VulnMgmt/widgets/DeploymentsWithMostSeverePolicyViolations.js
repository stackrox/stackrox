import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import { useQuery } from 'react-apollo';
import gql from 'graphql-tag';

import ViewAllButton from 'Components/ViewAllButton';
import Loader from 'Components/Loader';
import Widget from 'Components/Widget';
import NumberedList from 'Components/NumberedList';
import LabelChip from 'Components/LabelChip';
import NoResultsMessage from 'Components/NoResultsMessage';
import entityTypes from 'constants/entityTypes';
import workflowStateContext from 'Containers/workflowStateContext';
import queryService from 'modules/queryService';
import { getPolicySeverityCounts, sortDeploymentsByPolicyViolations } from 'utils/policyUtils';

const DEPLOYMENTS_WITH_MOST_SEVERE_POLICY_VIOLATIONS = gql`
    query deploymentsWithMostSeverePolicyViolations(
        $query: String
        $policyQuery: String
        $pagination: Pagination
    ) {
        results: deployments(query: $query, pagination: $pagination) {
            id
            name
            clusterName
            namespaceName: namespace
            failingPolicies(query: $policyQuery) {
                id
                severity
            }
        }
    }
`;

const processData = (data, workflowState, limit) => {
    const results = data.results.map(deployment => {
        const policySeverityCounts = getPolicySeverityCounts(deployment.failingPolicies);

        return { ...deployment, policySeverityCounts };
    });

    // @TODO, remove the chained .slice() call after backend pagination is available
    const sortedDeployments = sortDeploymentsByPolicyViolations(results).slice(0, limit);

    return sortedDeployments.map(
        ({ id, name, clusterName, namespaceName, policySeverityCounts }) => {
            const text = name;
            const { critical, high, medium, low } = policySeverityCounts;
            const tooltipTitle = name;
            const tooltipSubtitle = `${clusterName} / ${namespaceName}`;
            const tooltipBody = (
                <ul className="flex-1 border-base-300 overflow-hidden">
                    <li className="py-1 flex flex-col" key="description">
                        <span className="text-base-600 font-700 mr-2">Failing Policies:</span>
                        <span className="font-600">{`Critical: ${critical}`}</span>
                        <span className="font-600">{`High: ${high}`}</span>
                        <span className="font-600">{`Medium: ${medium}`}</span>
                        <span className="font-600">{`Low: ${low}`}</span>
                    </li>
                </ul>
            );
            return {
                text,
                url: workflowState.pushRelatedEntity(entityTypes.DEPLOYMENT, id).toUrl(),
                component: (
                    <>
                        <div className="mr-4">
                            <LabelChip text={`${low} L`} type="base" size="small" fade={!low} />
                        </div>
                        <div className="mr-4">
                            <LabelChip
                                text={`${medium} M`}
                                type="warning"
                                size="small"
                                fade={!medium}
                            />
                        </div>
                        <div className="mr-4">
                            <LabelChip
                                text={`${high} H`}
                                type="caution"
                                size="small"
                                fade={!high}
                            />
                        </div>
                        <LabelChip
                            text={`${critical} C`}
                            type="alert"
                            size="small"
                            fade={!critical}
                        />
                    </>
                ),
                tooltip: {
                    title: tooltipTitle,
                    subtitle: tooltipSubtitle,
                    body: tooltipBody
                }
            };
        }
    );
};

const DeploymentsWithMostSeverePolicyViolations = ({ entityContext, limit }) => {
    const { loading, data = {}, error } = useQuery(DEPLOYMENTS_WITH_MOST_SEVERE_POLICY_VIOLATIONS, {
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
        // @TODO: re-enable sorting again, after these fields are available for sorting in back-end pagination
        // .setSort([
        //     { id: 'failingPolicyCount', desc: true },
        //     { id: 'name', desc: false }
        // ])
        .toUrl();

    if (!loading && !error) {
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
