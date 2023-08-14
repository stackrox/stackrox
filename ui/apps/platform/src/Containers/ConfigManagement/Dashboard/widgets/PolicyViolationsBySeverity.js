import React, { useContext } from 'react';
import URLService from 'utils/URLService';
import ReactRouterPropTypes from 'react-router-prop-types';
import Widget from 'Components/Widget';
import Sunburst from 'Components/visuals/Sunburst';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import networkStatuses from 'constants/networkStatuses';
import { Link, withRouter } from 'react-router-dom';
import { gql } from '@apollo/client';
import max from 'lodash/max';
import { severityValues, severities } from 'constants/severities';
import { policySeverityColorMap } from 'constants/severityColors';
import { severityLabels as policySeverityLabels } from 'messages/common';
import { policySeverities } from 'types/policy.proto';
import policyStatus from 'constants/policyStatus';
import entityTypes from 'constants/entityTypes';
import searchContext from 'Containers/searchContext';
import { CLIENT_SIDE_SEARCH_OPTIONS as SEARCH_OPTIONS } from 'constants/searchOptions';
import { getPercentage } from 'utils/mathUtils';

const legendData = policySeverities.map((severity) => ({
    title: policySeverityLabels[severity],
    color: policySeverityColorMap[severity],
}));

const linkColor = 'var(--base-600)';
const textColor = 'var(--base-600)';
const passingChartColor = 'var(--base-400)';

const QUERY = gql`
    query policyViolationsBySeverity($query: String) {
        policies(query: $query) {
            id
            name
            categories
            severity
            disabled
            description
            lifecycleStages
            policyStatus
        }
    }
`;

function getCategorySeverity(category, violationsByCategory) {
    const maxSeverityValue = max(
        violationsByCategory[category]
            .filter((violation) => !violation.passing)
            .map((violation) => severityValues[violation.severity])
    );

    const severityEntry = Object.entries(severityValues).find(
        (entry) => entry[1] === maxSeverityValue
    );

    if (!severityEntry) {
        return passingChartColor;
    }

    return policySeverityColorMap[severityEntry[0]];
}

const PolicyViolationsBySeverity = ({ match, location }) => {
    const searchParam = useContext(searchContext);
    const processData = (data) => {
        if (!data || !data.policies || !data.policies.length) {
            return [];
        }
        return data.policies;
    };

    function getSunburstData(policies) {
        const violationsByCategory = policies.reduce((categories, policy) => {
            const { categories: policyCategories, severity, name: policyName } = policy;
            const isPassing = policy.policyStatus.toLowerCase() === policyStatus.PASS.toLowerCase();
            const newItems = { ...categories };
            policyCategories.forEach((category, idx) => {
                if (!newItems[category]) {
                    newItems[category] = [];
                }
                const color = !isPassing ? policySeverityColorMap[severity] : passingChartColor;
                const queryObj = !isPassing
                    ? {
                          [searchParam]: {
                              [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus.FAIL,
                          },
                      }
                    : null;
                const link = URLService.getURL(match, location)
                    .base(entityTypes.POLICY, policy.id)
                    .push(entityTypes.DEPLOYMENT)
                    .query(queryObj)
                    .url();

                const fullPolicyName = idx > 0 ? `${idx}. ${policyName}` : policyName;
                newItems[category].push({
                    severity,
                    passing: isPassing,
                    color,
                    textColor,
                    value: 0,
                    labelColor: color,
                    name: `${isPassing ? '' : 'View deployments violating'} "${fullPolicyName}"`,
                    link,
                });
            });
            return newItems;
        }, {});

        return Object.entries(violationsByCategory).map((entry) => {
            const category = entry[0];
            const children = entry[1];
            const numPassing = children.filter((child) => child.passing).length;
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
                textColor,
            };
        });
    }

    function getCenterValue(data) {
        const policiesInViolation = data.filter(
            (policy) => policy.policyStatus.toLowerCase() === 'fail'
        ).length;
        return policiesInViolation;
    }

    function getSummaryData(data) {
        const policiesInViolation = data.filter((policy) => policy.policyStatus === 'fail');
        function getCount(severity) {
            return policiesInViolation.filter((policy) => policy.severity === severity).length;
        }

        const criticalCount = getCount(severities.CRITICAL_SEVERITY);
        const highCount = getCount(severities.HIGH_SEVERITY);
        const mediumCount = getCount(severities.MEDIUM_SEVERITY);
        const lowCount = getCount(severities.LOW_SEVERITY);
        const passingCount =
            data.filter((policy) => !policy.disabled).length - policiesInViolation.length;

        const links = [];

        const url = URLService.getURL(match, location).base(entityTypes.POLICY);

        if (criticalCount) {
            links.push({
                text: `${criticalCount} rated as critical`,
                color: linkColor,
                link: url
                    .query({
                        [searchParam]: {
                            Severity: severities.CRITICAL_SEVERITY,
                            [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus.FAIL,
                        },
                    })
                    .url(),
            });
        }

        url.query(null);

        if (highCount) {
            links.push({
                text: `${highCount} rated as high`,
                color: linkColor,
                link: url
                    .query({
                        [searchParam]: {
                            Severity: severities.HIGH_SEVERITY,
                            Disabled: 'False',
                            [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus.FAIL,
                        },
                    })
                    .url(),
            });
        }

        url.query(null);

        if (mediumCount) {
            links.push({
                text: `${mediumCount} rated as medium`,
                color: linkColor,
                link: url
                    .query({
                        [searchParam]: {
                            Severity: severities.MEDIUM_SEVERITY,
                            Disabled: 'False',
                            [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus.FAIL,
                        },
                    })
                    .url(),
            });
        }

        url.query(null);

        if (lowCount) {
            links.push({
                text: `${lowCount} rated as low`,
                color: linkColor,
                link: url
                    .query({
                        [searchParam]: {
                            Severity: severities.LOW_SEVERITY,
                            Disabled: 'False',
                            [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus.FAIL,
                        },
                    })
                    .url(),
            });
        }

        url.query(null);

        if (passingCount) {
            links.push({
                text: `${passingCount} policies without violations`,
                color: linkColor,
                link: url
                    .query({
                        [searchParam]: {
                            Disabled: 'False',
                            [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus.PASS,
                        },
                    })
                    .url(),
            });
        }

        return links;
    }

    return (
        <Query
            query={QUERY}
            fetchPolicy="network-only"
            variables={{ query: 'LifeCycle Stage:DEPLOY' }}
        >
            {({ loading, data, networkStatus }) => {
                let contents = <Loader />;
                let viewAllLink = null;
                if (!loading && data && networkStatus === networkStatuses.READY) {
                    const filteredData = processData(data);
                    const sunburstData = getSunburstData(filteredData);
                    const centerValue = getCenterValue(filteredData);
                    const sidePanelData = getSummaryData(filteredData);

                    const linkTo = URLService.getURL(match, location)
                        .base(entityTypes.POLICY)
                        .url();

                    viewAllLink = (
                        <Link to={linkTo} className="no-underline btn-sm btn-base">
                            View all
                        </Link>
                    );

                    if (!sunburstData.length) {
                        contents = (
                            <div className="flex flex-1 items-center justify-center p-4 leading-loose">
                                No data available.
                            </div>
                        );
                    } else {
                        contents = (
                            <Sunburst
                                data={sunburstData}
                                rootData={sidePanelData}
                                legendData={legendData}
                                totalValue={centerValue}
                                units="value"
                            />
                        );
                    }
                }
                return (
                    <Widget
                        className="s-2 pdf-page"
                        header="Policy violations by severity"
                        headerComponents={viewAllLink}
                    >
                        {contents}
                    </Widget>
                );
            }}
        </Query>
    );
};

PolicyViolationsBySeverity.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
};

export default withRouter(PolicyViolationsBySeverity);
