import React, { useContext } from 'react';
import URLService from 'modules/URLService';
import ReactRouterPropTypes from 'react-router-prop-types';
import Widget from 'Components/Widget';
import Sunburst from 'Components/visuals/Sunburst';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import networkStatuses from 'constants/networkStatuses';
import { Link, withRouter } from 'react-router-dom';
import gql from 'graphql-tag';
import max from 'lodash/max';
import { severityValues, severities } from 'constants/severities';
import {
    severityColorMap,
    severityTextColorMap,
    severityColorLegend
} from 'constants/severityColors';
import policyStatus from 'constants/policyStatus';
import entityTypes from 'constants/entityTypes';
import searchContext from 'Containers/searchContext';
import { CLIENT_SIDE_SEARCH_OPTIONS as SEARCH_OPTIONS } from 'constants/searchOptions';
import { getPercentage } from 'utils/mathUtils';

const passingLinkColor = 'var(--base-500)';
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
            .filter(violation => !violation.passing)
            .map(violation => severityValues[violation.severity])
    );

    const severityEntry = Object.entries(severityValues).find(
        entry => entry[1] === maxSeverityValue
    );

    if (!severityEntry) return passingChartColor;

    return severityColorMap[severityEntry[0]];
}

const PolicyViolationsBySeverity = ({ match, location }) => {
    const searchParam = useContext(searchContext);
    const processData = data => {
        if (!data || !data.policies || !data.policies.length) return [];
        return data.policies;
    };

    function getSunburstData(policies) {
        const violationsByCategory = policies.reduce((categories, policy) => {
            const { categories: policyCategories, severity, name: policyName } = policy;
            const isPassing = policy.policyStatus.toLowerCase() === policyStatus.PASS.toLowerCase();
            const newItems = { ...categories };
            policyCategories.forEach((category, idx) => {
                if (!newItems[category]) newItems[category] = [];
                const color = !isPassing ? severityColorMap[severity] : passingChartColor;
                const queryObj = !isPassing
                    ? {
                          [searchParam]: {
                              [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus.FAIL
                          }
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
                    textColor: passingLinkColor,
                    value: 0,
                    labelColor: color,
                    name: `${isPassing ? '' : 'View deployments violating'} "${fullPolicyName}"`,
                    link
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

    function getCenterValue(data) {
        const policiesInViolation = data.filter(
            policy => policy.policyStatus.toLowerCase() === 'fail'
        ).length;
        return policiesInViolation;
    }

    function getSummaryData(data) {
        const policiesInViolation = data.filter(policy => policy.policyStatus === 'fail');
        function getCount(severity) {
            return policiesInViolation.filter(policy => policy.severity === severity).length;
        }

        const criticalCount = getCount(severities.CRITICAL_SEVERITY);
        const highCount = getCount(severities.HIGH_SEVERITY);
        const mediumCount = getCount(severities.MEDIUM_SEVERITY);
        const lowCount = getCount(severities.LOW_SEVERITY);
        const passingCount = data.length - policiesInViolation.length;

        const links = [];

        const url = URLService.getURL(match, location).base(entityTypes.POLICY);

        if (criticalCount)
            links.push({
                text: `${criticalCount} rated as critical`,
                color: severityTextColorMap.CRITICAL_SEVERITY,
                link: url
                    .query({
                        [searchParam]: {
                            Severity: severities.CRITICAL_SEVERITY,
                            [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus.FAIL
                        }
                    })
                    .url()
            });

        url.query(null);

        if (highCount)
            links.push({
                text: `${highCount} rated as high`,
                color: severityTextColorMap.HIGH_SEVERITY,
                link: url
                    .query({
                        [searchParam]: {
                            Severity: severities.HIGH_SEVERITY,
                            [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus.FAIL
                        }
                    })
                    .url()
            });

        url.query(null);

        if (mediumCount)
            links.push({
                text: `${mediumCount} rated as medium`,
                color: severityTextColorMap.MEDIUM_SEVERITY,
                link: url
                    .query({
                        [searchParam]: {
                            Severity: severities.MEDIUM_SEVERITY,
                            [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus.FAIL
                        }
                    })
                    .url()
            });

        url.query(null);

        if (lowCount)
            links.push({
                text: `${lowCount} rated as low`,
                color: severityTextColorMap.LOW_SEVERITY,
                link: url
                    .query({
                        [searchParam]: {
                            Severity: severities.LOW_SEVERITY,
                            [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus.FAIL
                        }
                    })
                    .url()
            });

        url.query(null);

        if (passingCount)
            links.push({
                text: `${passingCount} policies without violations`,
                color: passingLinkColor,
                link: url
                    .query({
                        [searchParam]: {
                            [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus.PASS
                        }
                    })
                    .url()
            });

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
                        <Link to={linkTo} className="no-underline">
                            <button className="btn-sm btn-base" type="button">
                                View All
                            </button>
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
                                legendData={severityColorLegend}
                                totalValue={centerValue}
                                units="value"
                            />
                        );
                    }
                }
                return (
                    <Widget
                        className="s-2 pdf-page"
                        header="Policy Violations by Severity"
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
    location: ReactRouterPropTypes.location.isRequired
};

export default withRouter(PolicyViolationsBySeverity);
