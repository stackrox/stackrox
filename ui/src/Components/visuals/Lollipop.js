import React from 'react';
import {
    FlexibleWidthXYPlot,
    XAxis,
    YAxis,
    VerticalGridLines,
    HorizontalBarSeries,
    LabelSeries,
    GradientDefs
} from 'react-vis';
import { max } from 'lodash';
import { withRouter } from 'react-router-dom';
import PropTypes from 'prop-types';
import BarGradient from './BarGradient';

const Lollipop = ({ data }) => {
    function getGridLineValues() {
        const interval = data.length < 5 ? 1 : 5;
        const values = data.map(datum => datum.x);
        const maxVal = Math.round(max(values) / interval) * interval;
        const lineValues = [];
        for (let x = 0; x <= maxVal + interval; x += interval) {
            lineValues.push(x);
        }
        return lineValues;
    }
    const gridLineValues = getGridLineValues();

    function formatTick(value) {
        return Math.round(value);
    }

    function getLabelData() {
        return data.map((item, index) => {
            const val = {
                y: item.y,
                yOffset: -25,
                xOffset: 10,
                label: ` ${index + 1}. ${item.y}`
            };
            return val;
        });
    }
    const labelData = getLabelData();

    return (
        <div className="relative chart-container w-full horizontal-bar-responsive">
            <FlexibleWidthXYPlot
                height={350}
                yType="category"
                yRange={data.map((item, i) => (i + 1) * 41).concat([0])}
                margin={{ top: 33.3, left: 7 }}
                stackBy="x"
                xDomain={[0, max(gridLineValues)]}
            >
                <GradientDefs>
                    <BarGradient />
                </GradientDefs>
                <VerticalGridLines tickValues={gridLineValues} />

                <XAxis
                    orientation="top"
                    tickSize={0}
                    tickValues={gridLineValues}
                    tickFormat={formatTick}
                />

                <HorizontalBarSeries
                    data={data}
                    style={{
                        height: 3,
                        rx: '2px',
                        cursor: 'pointer'
                    }}
                    color="url(#horizontalGradient)"
                />
                <LabelSeries
                    data={labelData}
                    className="text-xs pointer-events-none theme-light"
                    labelAnchorX="start-alignment"
                    labelAnchorY="baseline"
                    style={{
                        fill: 'var(--primary-800)',
                        cursor: 'pointer',
                        transform: 'translate(15px,35px)'
                    }}
                />
                {/* Todo: create label series for the lollipip head */}

                <YAxis tickSize={0} top={26} className="text-xs" />
            </FlexibleWidthXYPlot>
        </div>
    );
};

Lollipop.propTypes = {
    data: PropTypes.shape({}).isRequired
};

export default withRouter(Lollipop);
