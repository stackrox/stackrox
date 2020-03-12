import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import gql from 'graphql-tag';
import { useQuery } from 'react-apollo';
import max from 'lodash/max';

import queryService from 'modules/queryService';
import Widget from 'Components/Widget';
import Sunburst from 'Components/visuals/Sunburst';
import Loader from 'Components/Loader';
import ViewAllButton from 'Components/ViewAllButton';
import workflowStateContext from 'Containers/workflowStateContext';
import { severityValues, severities } from 'constants/severities';

import { severityColorMap, severityTextColorMap } from 'constants/severityColors';
import policyStatus from 'constants/policyStatus';
import entityTypes from 'constants/entityTypes';
import { CLIENT_SIDE_SEARCH_OPTIONS as SEARCH_OPTIONS } from 'constants/searchOptions';
import { getPercentage } from 'utils/mathUtils';
import NoResultsMessage from 'Components/NoResultsMessage';

const passingLinkColor = 'var(--base-500)';
const passingChartColor = 'var(--base-400)';

const POLICIES_QUERY = gql`
    query policyViolationsBySeverity($query: String, $policyQuery: String) {
        deployments(query: $query) {
            id
            name
            failingPolicies(query: $policyQuery) {
                id
                name
                categories
                description
                policyStatus(query: $query)
                lastUpdated
                latestViolation(query: $query)
                severity
                deploymentCount(query: $query)
                lifecycleStages
                enforcementActions
                notifiers
            }
        }
    }
`;

function getCategorySeverity(category, violationsByCategory) {
    const maxSeverityValue = max(
        violationsByCategory[category]
            .filter(violation => !violation.passing)
            .map(violation => severityValues[violation.severity])
    );

    const severityEntry = Object.entries(severityValues).find(
        entry => entry[1] === maxSeverityValue
    );

    if (!severityEntry) return passingChartColor;

    return severityColorMap[severityEntry[0]];
}

const PolicyViolationsBySeverity = ({ entityContext }) => {
    const { loading, data = {} } = useQuery(POLICIES_QUERY, {
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
        .pushList(entityTypes.POLICY)
        .setSort([{ id: 'policyStatus', desc: true }, { id: 'severity', desc: true }])
        .toUrl();

    if (!loading) {
        if (
            !data ||
            !data.deployments ||
            !data.deployments.length ||
            !data.deployments[0].failingPolicies
        ) {
            content = (
                <div className="flex mx-auto items-center">No scanner setup for this registry.</div>
            );
        } else if (!data.deployments[0].failingPolicies.length) {
            content = (
                <NoResultsMessage
                    message="This deployment does not violate any active system policies"
                    className="p-6"
                    icon="info"
                />
            );
        } else {
            const filteredData = processData(data.deployments[0].failingPolicies);
            const sunburstData = getSunburstData(filteredData);
            const centerValue = getCenterValue(filteredData);
            const sidePanelData = getSummaryData(filteredData);

            if (!sunburstData.length) {
                content = (
                    <NoResultsMessage
                        message="No policy violations found"
                        className="p-6"
                        icon="info"
                    />
                );
            } else {
                content = (
                    <Sunburst
                        data={sunburstData}
                        rootData={sidePanelData}
                        totalValue={centerValue}
                        units="value"
                        small
                    />
                );
            }
        }
    }

    function processData(policies) {
        if (!policies || !policies.length) return [];
        return policies;
    }

    function getSunburstData(policies) {
        const violationsByCategory = policies.reduce((categories, policy) => {
            const { categories: policyCategories, severity, name: policyName } = policy;
            const isPassing = policy.policyStatus.toLowerCase() === policyStatus.PASS.toLowerCase();
            const newItems = { ...categories };
            policyCategories.forEach((category, idx) => {
                if (!newItems[category]) newItems[category] = [];
                const color = !isPassing ? severityColorMap[severity] : passingChartColor;

                const fullPolicyName = idx > 0 ? `${idx}. ${policyName}` : policyName;
                newItems[category].push({
                    severity,
                    passing: isPassing,
                    color,
                    textColor: passingLinkColor,
                    value: 0,
                    labelColor: color,
                    name: `${isPassing ? '' : 'View deployments violating'} "${fullPolicyName}"`
                });
            });
            return newItems;
        }, {});

        return Object.entries(violationsByCategory).map(entry => {
            const category = entry[0];
            const children = entry[1];
            const numPassing = children.filter(child => child.passing).length;
            const labelValue = `${children.length - numPassing}/${
                children.length
            } policies violated`;
            const value = getPercentage(numPassing, children.length);
            const color = getCategorySeverity(category, violationsByCategory);
            return {
                name: category,
                children,
                value,
                labelValue,
                color,
                textColor: passingLinkColor
            };
        });
    }

    function getCenterValue(rawData) {
        const policiesInViolation = rawData.filter(
            policy => policy.policyStatus.toLowerCase() === 'fail'
        ).length;
        return policiesInViolation;
    }

    function getSummaryData(rawData) {
        const policiesInViolation = rawData.filter(policy => policy.policyStatus === 'fail');
        function getCount(severity) {
            return policiesInViolation.filter(policy => policy.severity === severity).length;
        }

        // @TODO: check with back-end to see if these counts can be calculated there
        const criticalCount = getCount(severities.CRITICAL_SEVERITY);
        const highCount = getCount(severities.HIGH_SEVERITY);
        const mediumCount = getCount(severities.MEDIUM_SEVERITY);
        const lowCount = getCount(severities.LOW_SEVERITY);
        const passingCount = rawData.length - policiesInViolation.length;

        const links = [];

        const newState = workflowState.resetPage(entityTypes.POLICY);
        if (criticalCount)
            links.push({
                text: `${criticalCount} rated as critical`,
                color: severityTextColorMap.CRITICAL_SEVERITY,
                link: newState
                    .setSearch({
                        Severity: severities.CRITICAL_SEVERITY,
                        [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus.FAIL
                    })
                    .toUrl()
            });

        if (highCount)
            links.push({
                text: `${highCount} rated as high`,
                color: severityTextColorMap.HIGH_SEVERITY,
                link: newState
                    .setSearch({
                        Severity: severities.HIGH_SEVERITY,
                        [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus.FAIL
                    })
                    .toUrl()
            });

        if (mediumCount)
            links.push({
                text: `${mediumCount} rated as medium`,
                color: severityTextColorMap.MEDIUM_SEVERITY,
                link: newState
                    .setSearch({
                        Severity: severities.MEDIUM_SEVERITY,
                        [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus.FAIL
                    })
                    .toUrl()
            });

        if (lowCount)
            links.push({
                text: `${lowCount} rated as low`,
                color: severityTextColorMap.LOW_SEVERITY,
                link: newState
                    .setSearch({
                        Severity: severities.LOW_SEVERITY,
                        [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus.FAIL
                    })
                    .toUrl()
            });

        if (passingCount)
            links.push({
                text: `${passingCount} policies without violations`,
                color: passingLinkColor,
                link: newState
                    .setSearch({
                        [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus.PASS
                    })
                    .toUrl()
            });

        return links;
    }

    return (
        <Widget
            className="s-2 pdf-page h-full"
            header="Policy Violations by Severity"
            headerComponents={<ViewAllButton url={viewAllURL} />}
        >
            {content}
        </Widget>
    );
};

PolicyViolationsBySeverity.propTypes = {
    entityContext: PropTypes.shape({}),
    policyContext: PropTypes.shape({})
};

PolicyViolationsBySeverity.defaultProps = {
    entityContext: {},
    policyContext: {}
};

export default PolicyViolationsBySeverity;
