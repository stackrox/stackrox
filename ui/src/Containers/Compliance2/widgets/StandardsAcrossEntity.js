import React from 'react';
import PropTypes from 'prop-types';

import Widget from 'Components/Widget';
import HorizontalBarChart from 'Components/visuals/HorizontalBar';
import Query from 'Components/ThrowingQuery';
import { CLUSTERS_QUERY } from 'queries/cluster';
import { horizontalBarData } from 'mockData/graphDataMock';
import { withRouter } from 'react-router-dom';
import Loader from 'Components/Loader';

function formatAsPercent(x) {
    return `${x}%`;
}

const StandardsAcrossEntity = ({ type }) => (
    // TODO: use real query and calculate values based on return data
    <Query query={CLUSTERS_QUERY} action="list">
        {({ loading, data }) => {
            let graphData;
            let contents = <Loader />;

            if (!loading && data) {
                graphData = horizontalBarData;
                contents = <HorizontalBarChart data={graphData} valueFormat={formatAsPercent} />;
            }

            return (
                <Widget header={`Standards Across ${type}`} bodyClassName="p-2">
                    {contents}
                </Widget>
            );
        }}
    </Query>
);

StandardsAcrossEntity.propTypes = {
    type: PropTypes.string.isRequired,
    match: PropTypes.shape({}).isRequired
};

export default withRouter(StandardsAcrossEntity);
