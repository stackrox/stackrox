import React from 'react';
import componentTypes from 'constants/componentTypes';
import { resourceTypes } from 'constants/entityTypes';

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

function processData(data) {
    const { complianceStandards } = data;
    if (!complianceStandards) return [];
    const barData = data.results.results.map(result => {
        const standard = complianceStandards.find(cs => cs.id === result.aggregationKeys[0].id);
        const { numPassing, numFailing } = result;
        const percentagePassing = Math.round((numPassing / (numFailing + numPassing)) * 100) || 0;
        const dataPoint = {
            y: standard.name,
            x: percentagePassing,
            hint: {
                title: `${standard.name} Standard - ${percentagePassing}%`,
                body: `[] failing across ${numFailing + numPassing} clusters`
            },
            axisLink: `/main/compliance2/${standard.id}`
        };
        return dataPoint;
    });
    return barData;
}

const StandardsAcrossEntity = ({ type, params }) => (
    <Query params={params} componentType={componentTypeMapping[type]} pollInterval={5000}>
        {({ loading, data }) => {
            let contents;
            const headerText = `Standards Across ${type}`;
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
                <Widget header={headerText} bodyClassName="p-2">
                    {contents}
                </Widget>
            );
        }}
    </Query>
);

StandardsAcrossEntity.propTypes = {
    type: PropTypes.string.isRequired,
    params: PropTypes.shape({}).isRequired
};

export default StandardsAcrossEntity;
