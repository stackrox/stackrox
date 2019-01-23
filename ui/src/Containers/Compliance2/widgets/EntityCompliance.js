import React from 'react';
import PropTypes from 'prop-types';
import Widget from 'Components/Widget';
import { verticalSingleBarData } from 'mockData/graphDataMock';
import VerticalBarChart from 'Components/visuals/VerticalBar';
import ArcSingle from 'Components/visuals/ArcSingle';

const EntityCompliance = ({ type }) => (
    <Widget header={`${type} Compliance`} className="bg-base-100 sx-2 sy-1">
        <div className="flex w-full" style={{ alignItems: 'center' }}>
            <div className="p-2">
                <ArcSingle value={78} />
            </div>
            <div className="flex-grow -m-2">
                <VerticalBarChart
                    plotProps={{ height: 180, width: 250 }}
                    data={verticalSingleBarData}
                />
            </div>
        </div>
    </Widget>
);

EntityCompliance.propTypes = {
    type: PropTypes.string.isRequired
};

export default EntityCompliance;
