import React from 'react';
import gql from 'graphql-tag';
import Loader from 'Components/Loader';
import { Link, withRouter } from 'react-router-dom';
import URLService from 'modules/URLService';
import networkStatuses from 'constants/networkStatuses';
import Query from 'Components/ThrowingQuery';
import Widget from 'Components/Widget';
import entityTypes from 'constants/entityTypes';
import ReactRouterPropTypes from 'react-router-prop-types';
import severityColorMap from 'constants/severityColors';
import { Tooltip } from 'react-tippy';
import pluralize from 'pluralize';
import NoResultsMessage from 'Components/NoResultsMessage';

const severityFontMap = {
    CRITICAL_SEVERITY: 'var(--alert-800)',
    HIGH_SEVERITY: 'var(--caution-800)',
    MEDIUM_SEVERITY: 'var(--warning-800)',
    LOW_SEVERITY: 'var(--base-800)'
};

const severityTexts = {
    CRITICAL_SEVERITY: 'Critical',
    HIGH_SEVERITY: 'High',
    MEDIUM_SEVERITY: 'Medium',
    LOW_SEVERITY: 'Low'
};

const QUERY = gql`
    query deployments {
        deployments {
            id
            name
            deployAlerts {
                policy {
                    severity
                }
            }
        }
    }
`;

const PolicyViolationsByDeployment = ({ match, location }) => {
    function processData(data) {
        if (!data || !data.deployments) return [];

        const results = data.deployments.map(deployment => {
            const counts = deployment.deployAlerts.reduce(
                (total, alert) => {
                    const ret = { ...total };
                    ret[alert.policy.severity] += 1;
                    return ret;
                },
                {
                    CRITICAL_SEVERITY: 0,
                    HIGH_SEVERITY: 0,
                    MEDIUM_SEVERITY: 0,
                    LOW_SEVERITY: 0
                }
            );
            return {
                id: deployment.id,
                name: deployment.name,
                counts
            };
        });

        function score(counts) {
            return (
                counts.CRITICAL_SEVERITY * 1000 +
                counts.HIGH_SEVERITY * 100 +
                counts.MEDIUM_SEVERITY * 10 +
                counts.LOW_SEVERITY
            );
        }
        results.sort((a, b) => {
            return score(b.counts) - score(a.counts);
        });

        return results;
    }

    return (
        <Query query={QUERY}>
            {({ loading, data, networkStatus }) => {
                let contents = <Loader />;
                let viewAllLink;

                if (!loading && data && networkStatus === networkStatuses.READY) {
                    const linkTo = URLService.getURL(match, location)
                        .base(entityTypes.DEPLOYMENT)
                        .url();

                    viewAllLink = (
                        <Link to={linkTo} className="no-underline">
                            <button className="btn-sm btn-base" type="button">
                                View All
                            </button>
                        </Link>
                    );

                    const deploymentsWithAlerts = data.deployments.filter(
                        datum => datum.deployAlerts.length
                    );
                    if (!deploymentsWithAlerts.length) {
                        contents = <NoResultsMessage message="No deployments with violations" />;
                    } else {
                        const results = processData(deploymentsWithAlerts);
                        const slicedData = results.slice(0, 10);

                        contents = (
                            <ul className="list-reset w-full columns-2 columns-gap-0 text-sm">
                                {slicedData.map((item, index) => (
                                    <Link
                                        key={`${item.id}-${index}`}
                                        to={`${linkTo}/${item.id}`}
                                        className="no-underline text-base-600"
                                    >
                                        <li className="hover:bg-base-200">
                                            <div
                                                className={`flex flex-row border-base-300 ${
                                                    index !== 4 && index !== 9 ? 'border-b' : ''
                                                } ${index < 5 ? 'border-r' : ''}`}
                                            >
                                                <div className="flex flex-col flex-1 truncate py-2 px-4">
                                                    <span className="pb-2">
                                                        {index + 1}. {item.name}
                                                    </span>
                                                    <ul className="list-reset flex">
                                                        {Object.keys(item.counts).map(type => {
                                                            const style = {
                                                                backgroundColor:
                                                                    severityColorMap[type],
                                                                color: severityFontMap[type],
                                                                borderColor: severityColorMap[type]
                                                            };
                                                            const count = item.counts[type];

                                                            if (count === 0) return null;

                                                            const tipText = `${item.counts[type]} ${
                                                                severityTexts[type]
                                                            } ${pluralize('Violation', count)}`;
                                                            return (
                                                                <Tooltip
                                                                    position="top"
                                                                    trigger="mouseenter"
                                                                    animation="none"
                                                                    duration={0}
                                                                    arrow
                                                                    distance={20}
                                                                    html={
                                                                        <span className="text-sm">
                                                                            {tipText}
                                                                        </span>
                                                                    }
                                                                    key={`${type}`}
                                                                    unmountHTMLWhenHide
                                                                >
                                                                    <li
                                                                        className="p-1 border rounded mr-2"
                                                                        style={style}
                                                                    >
                                                                        <span>{count} </span>
                                                                        <span className="uppercase">
                                                                            {type.charAt(0)}
                                                                        </span>
                                                                    </li>
                                                                </Tooltip>
                                                            );
                                                        })}
                                                    </ul>
                                                </div>
                                            </div>
                                        </li>
                                    </Link>
                                ))}
                            </ul>
                        );
                    }
                }
                return (
                    <Widget
                        className="s-2 overflow-hidden pdf-page"
                        header="Deployments with most severe policy violations"
                        headerComponents={viewAllLink}
                    >
                        {contents}
                    </Widget>
                );
            }}
        </Query>
    );
};

PolicyViolationsByDeployment.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired
};

export default withRouter(PolicyViolationsByDeployment);
