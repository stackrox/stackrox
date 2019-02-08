import React from 'react';
import componentTypes from 'constants/componentTypes';
import { resourceTypes } from 'constants/entityTypes';
import { resourceLabels } from 'messages/common';
import pluralize from 'pluralize';

import Widget from 'Components/Widget';
import Query from 'Components/AppQuery';
import Loader from 'Components/Loader';
import PropTypes from 'prop-types';
import HorizontalBarChart from 'Components/visuals/HorizontalBar';
import NoResultsMessage from 'Components/NoResultsMessage';

const componentTypeMapping = {
    [resourceTypes.CLUSTER]: componentTypes.STANDARDS_ACROSS_CLUSTERS,
    [resourceTypes.NAMESPACE]: componentTypes.STANDARDS_ACROSS_NAMESPACES,
    [resourceTypes.NODE]: componentTypes.STANDARDS_ACROSS_NODES
};

function formatAsPercent(x) {
    return `${x}%`;
}

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
        const dataPoint = {
            y: standard.name,
            x: percentagePassing,
            hint: {
                title: `${standard.name} Standard - ${percentagePassing}% Passing`,
                body: `${total - passing} failing checks across all ${pluralize(
                    resourceLabels[type]
                )}`
            },
            barLink: `/main/compliance2/${standard.id}?groupBy=${type}`,
            axisLink: `/main/compliance2/${standard.id}`
        };
        return dataPoint;
    });

    return barData;
}

const StandardsAcrossEntity = ({ type, params, pollInterval }) => (
    <Query params={params} componentType={componentTypeMapping[type]} pollInterval={pollInterval}>
        {({ loading, data }) => {
            let contents;
            const headerText = `Standards Across ${type}s`;
            if (!loading || data.complianceStandards) {
                const results = processData(data, type);
                if (!results.length) {
                    contents = <NoResultsMessage message="No Data Available" />;
                } else {
                    contents = <HorizontalBarChart data={results} valueFormat={formatAsPercent} />;
                }
            } else {
                contents = <Loader />;
            }
            return (
                <Widget className="sx-2 sy-2" header={headerText} bodyClassName="p-4">
                    {contents}
                </Widget>
            );
        }}
    </Query>
);

StandardsAcrossEntity.propTypes = {
    type: PropTypes.string.isRequired,
    params: PropTypes.shape({}).isRequired,
    pollInterval: PropTypes.number
};

StandardsAcrossEntity.defaultProps = {
    pollInterval: 0
};

export default StandardsAcrossEntity;
