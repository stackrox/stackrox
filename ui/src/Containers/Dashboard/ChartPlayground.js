import { connect } from 'react-redux';
import React from 'react';
// Chart stuff
import HorizontalBarChart from 'Components/visuals/HorizontalBar';
import Sunburst from 'Components/visuals/Sunburst';
import VerticalBarChart from 'Components/visuals/VerticalClusterBar';

import {
    horizontalBarData,
    sunburstData,
    sunburstLegendData,
    verticalBarData
} from 'mockData/graphDataMock';

function formatAsPercent(x) {
    return `${x}%`;
}

const ChartPlayground = () => (
    <div className="h-full overflow-scroll">
        <div className="flex w-full flex-wrap -mx-6 p-3">
            <div className="w-full lg:w-1/3 p-3">
                <VerticalBarChart
                    data={verticalBarData}
                    labelLinks={{
                        'Docker Swarm Dev': 'https://google.com/search?q=docker'
                    }}
                />
            </div>
            <div className="w-full lg:w-1/3 p-3">
                <HorizontalBarChart data={horizontalBarData} valueFormat={formatAsPercent} />
            </div>
            <div className="w-full lg:w-1/3 p-3">
                <Sunburst data={sunburstData} legendData={sunburstLegendData} centerLabel="75%" />
            </div>
        </div>
    </div>
);

export default connect()(ChartPlayground);
