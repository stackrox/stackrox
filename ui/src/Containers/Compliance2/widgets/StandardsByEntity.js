import React from 'react';
import PropTypes from 'prop-types';
import Widget from 'Components/Widget';
import { verticalBarData } from 'mockData/graphDataMock';
import VerticalBarChart from 'Components/visuals/VerticalClusterBar';

const StandardsByEntity = ({ type }) => {
    // Lets pretend we got data from a graphQL Query
    const data = verticalBarData;

    const VerticalBarChartPaged = ({ currentPage }) => (
        <>
            <VerticalBarChart
                data={data[currentPage]}
                labelLinks={{
                    'Docker Swarm Dev': 'https://google.com/search?q=docker'
                }}
            />
        </>
    );

    return (
        <Widget pages={data.length} header={`Standards By ${type}`} className="bg-base-100">
            <VerticalBarChartPaged />
        </Widget>
    );
};

StandardsByEntity.propTypes = {
    type: PropTypes.string.isRequired
};

export default StandardsByEntity;
