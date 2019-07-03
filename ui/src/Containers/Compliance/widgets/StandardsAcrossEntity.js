import React, { useContext } from 'react';
import entityTypes, { standardBaseTypes } from 'constants/entityTypes';
import { resourceLabels } from 'messages/common';
import { standardLabels } from 'messages/standards';
import URLService from 'modules/URLService';
import pluralize from 'pluralize';
import toLower from 'lodash/toLower';
import ReactRouterPropTypes from 'react-router-prop-types';
import merge from 'lodash/merge';

import Widget from 'Components/Widget';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PropTypes from 'prop-types';
import HorizontalBarChart from 'Components/visuals/HorizontalBar';
import NoResultsMessage from 'Components/NoResultsMessage';
import { withRouter } from 'react-router-dom';
import { AGGREGATED_RESULTS_ACROSS_ENTITY as QUERY } from 'queries/controls';
import searchContext from 'Containers/searchContext';

function formatAsPercent(x) {
    return `${x}%`;
}

function setStandardsMapping(data, type) {
    const mapping = {};
    data.results.forEach(result => {
        const standardId = result.aggregationKeys[0].id;
        const { numPassing, numFailing } = result;
        if (!mapping[standardId]) {
            mapping[standardId] = {};
            mapping[standardId][type] = {
                passing: numPassing,
                total: numPassing + numFailing
            };
        } else {
            const { passing, total } = mapping[standardId][type];
            mapping[standardId][type] = {
                passing: passing + numPassing,
                total: total + (numPassing + numFailing)
            };
        }
    });
    return mapping;
}

const StandardsAcrossEntity = ({ match, location, entityType, bodyClassName, className }) => {
    const searchParam = useContext(searchContext);

    function processData(data, type) {
        if (!data || !data.results || !data.results.results.length) return [];
        const { complianceStandards } = data;
        const standardsMapping = merge(
            {},
            setStandardsMapping(data.results, 'checks'),
            setStandardsMapping(data.controls, 'controls')
        );

        const barData = Object.keys(standardsMapping).map(standardId => {
            const standard = complianceStandards.find(cs => cs.id === standardId);
            const { controls, checks } = standardsMapping[standardId];
            const { passing: passingControls, total: totalControls } = controls;
            const { passing: passingChecks, total: totalChecks } = checks;
            const percentagePassing = Math.round((passingChecks / totalChecks) * 100) || 0;
            const link = URLService.getURL(match, location)
                .base(entityTypes.CONTROL)
                .query({
                    [searchParam]: {
                        groupBy: type,
                        Standard: standardLabels[standardId]
                    }
                })
                .url();
            const dataPoint = {
                y: standardBaseTypes[standardId],
                x: percentagePassing,
                hint: {
                    title: `${standard.name} Standard - ${percentagePassing}% Passing`,
                    body: `${totalControls -
                        passingControls} failing controls across all ${pluralize(
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
        unit: entityTypes.CHECK
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
