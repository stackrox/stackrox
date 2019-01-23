import React from 'react';
import PropTypes from 'prop-types';
import Widget from 'Components/Widget';
import { verticalBarData } from 'mockData/graphDataMock';
import VerticalBarChart from 'Components/visuals/VerticalClusterBar';
import Query from 'Components/ThrowingQuery';
import { CLUSTERS_QUERY } from 'queries/cluster';
import Loader from 'Components/Loader';
import { withRouter } from 'react-router-dom';

const StandardsByEntity = ({ type }) => (
    // TODO: use real query and calculate values based on return data
    <Query query={CLUSTERS_QUERY} action="list">
        {({ loading, data }) => {
            let graphData;
            let labelLinks;
            let pages;
            let contents = <Loader />;

            if (!loading && data) {
                graphData = verticalBarData;
                labelLinks = {
                    'Docker Swarm Dev': 'https://google.com/search?q=docker'
                };
                pages = verticalBarData.length;

                const VerticalBarChartPaged = ({ currentPage }) => (
                    <VerticalBarChart data={graphData[currentPage]} labelLinks={labelLinks} />
                );
                VerticalBarChartPaged.propTypes = { currentPage: PropTypes.number };
                VerticalBarChartPaged.defaultProps = { currentPage: 0 };
                contents = <VerticalBarChartPaged />;
            }

            return (
                <Widget pages={pages} header={`Standards By ${type}`}>
                    {contents}
                </Widget>
            );
        }}
    </Query>
);
StandardsByEntity.propTypes = {
    type: PropTypes.string.isRequired
};

export default withRouter(StandardsByEntity);
