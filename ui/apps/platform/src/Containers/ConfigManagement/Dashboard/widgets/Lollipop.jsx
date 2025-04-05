import React from 'react';
import {
    FlexibleWidthXYPlot,
    XAxis,
    YAxis,
    VerticalGridLines,
    HorizontalBarSeries,
    MarkSeries,
    LabelSeries,
    GradientDefs,
} from 'react-vis';
import max from 'lodash/max';
import { useNavigate } from 'react-router-dom';
import PropTypes from 'prop-types';
import BarGradient from 'Components/visuals/BarGradient';

const Lollipop = ({ data }) => {
    const navigate = useNavigate();

    function getGridLineValues() {
        const interval = data.length < 5 ? 1 : 5;
        const values = data.map((datum) => datum.x);
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
                link: item.link,
                x: null,
                y: item.y,
                yOffset: -25,
                xOffset: 10,
                label: ` ${index + 1}. ${item.y}`,
            };
            return val;
        });
    }

    function onValueClickHandler(datum) {
        if (datum.link) {
            navigate(datum.link);
        }
    }

    const labelData = getLabelData();
    const endcapData = [...data];

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
                        cursor: 'pointer',
                    }}
                    color="url(#horizontalGradient)"
                    onValueClick={onValueClickHandler}
                    stack
                />
                <MarkSeries
                    data={endcapData}
                    marginTop="17"
                    color="#BDF3FF"
                    onValueClick={onValueClickHandler}
                />
                <LabelSeries
                    data={labelData}
                    labelAnchorX="start-alignment"
                    labelAnchorY="baseline"
                    onValueClick={onValueClickHandler}
                    style={{
                        fill: 'var(--pf-v5-global--link--Color)',
                        cursor: 'pointer',
                        transform: 'translate(15px,35px)',
                    }}
                />

                <YAxis tickSize={0} top={26} className="text-xs" />
            </FlexibleWidthXYPlot>
        </div>
    );
};

Lollipop.propTypes = {
    data: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
};

export default Lollipop;
