import React from 'react';
import entityTypes, { standardBaseTypes } from 'constants/entityTypes';
import { resourceLabels } from 'messages/common';
import { standardLabels } from 'messages/standards';
import URLService from 'modules/URLService';
import pluralize from 'pluralize';
import toLower from 'lodash/toLower';
import ReactRouterPropTypes from 'react-router-prop-types';
import Widget from 'Components/Widget';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PropTypes from 'prop-types';
import HorizontalBarChart from 'Components/visuals/HorizontalBar';
import NoResultsMessage from 'Components/NoResultsMessage';
import { AGGREGATED_RESULTS as QUERY } from 'queries/controls';
import { withRouter } from 'react-router-dom';

function formatAsPercent(x) {
    return `${x}%`;
}

const StandardsAcrossEntity = ({ match, location, entityType, bodyClassName, className }) => {
    function processData(data, type) {
        if (!data || !data.results || !data.results.results.length) return [];
        const { complianceStandards } = data;
        const standardsMapping = {};

        data.results.results.forEach(result => {
            const standardId = result.aggregationKeys[0].id;
            const { numPassing, numFailing } = result;
            if (!standardsMapping[standardId]) {
                standardsMapping[standardId] = {
                    passing: numPassing,
                    total: numPassing + numFailing
                };
            } else {
                standardsMapping[standardId] = {
                    passing: standardsMapping[standardId].passing + numPassing,
                    total: standardsMapping[standardId].total + (numPassing + numFailing)
                };
            }
        });

        const barData = Object.keys(standardsMapping).map(standardId => {
            const standard = complianceStandards.find(cs => cs.id === standardId);
            const { passing, total } = standardsMapping[standardId];
            const percentagePassing = Math.round((passing / total) * 100) || 0;
            const link = URLService.getURL(match, location)
                .base(entityTypes.CONTROL)
                .query({
                    groupBy: type,
                    Standard: standardLabels[standardId]
                })
                .url();
            const dataPoint = {
                y: standardBaseTypes[standardId],
                x: percentagePassing,
                hint: {
                    title: `${standard.name} Standard - ${percentagePassing}% Passing`,
                    body: `${total - passing} failing controls across all ${pluralize(
                        resourceLabels[type]
                    )}`
                },
                link
            };
            return dataPoint;
        });

        return barData;
    }

    const variables = {
        groupBy: [entityTypes.STANDARD, entityType],
        unit: entityTypes.CONTROL
    };
    return (
        <Query query={QUERY} variables={variables}>
            {({ loading, data }) => {
                let contents;
                const headerText = `Passing standards across ${entityType}s`;
                if (!loading || data.complianceStandards) {
                    const results = processData(data, entityType);
                    if (!results.length) {
                        contents = (
                            <NoResultsMessage message="No data available. Please run a scan." />
                        );
                    } else {
                        contents = (
                            <HorizontalBarChart data={results} valueFormat={formatAsPercent} />
                        );
                    }
                } else {
                    contents = <Loader />;
                }
                return (
                    <Widget
                        className={`s-2 ${className}`}
                        header={headerText}
                        id={`standards-across-${toLower(entityType)}`}
                        bodyClassName={`graph-bottom-border ${bodyClassName}`}
                    >
                        {contents}
                    </Widget>
                );
            }}
        </Query>
    );
};

StandardsAcrossEntity.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    entityType: PropTypes.string.isRequired,
    bodyClassName: PropTypes.string,
    className: PropTypes.string
};

StandardsAcrossEntity.defaultProps = {
    bodyClassName: 'px-4 pt-1',
    className: ''
};

export default withRouter(StandardsAcrossEntity);
