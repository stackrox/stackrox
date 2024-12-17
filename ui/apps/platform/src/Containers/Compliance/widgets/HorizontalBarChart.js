import React from 'react';
import {
    FlexibleWidthXYPlot,
    XAxis,
    YAxis,
    VerticalGridLines,
    HorizontalBarSeries,
    LabelSeries,
} from 'react-vis';
import { Link, useNavigate } from 'react-router-dom';
import PropTypes from 'prop-types';
import merge from 'deepmerge';

import { getColor } from './colorsForCompliance';

const minimalMargin = { top: -15, bottom: 0, left: 0, right: 0 };

const sortByYValue = (a, b) => {
    if (a.y < b.y) {
        return -1;
    }
    if (a.y > b.y) {
        return 1;
    }
    return 0;
};

const HorizontalBarChart = ({
    data,
    containerProps = {},
    plotProps = {},
    seriesProps = {},
    valueFormat = (x) => x,
    tickValues = [0, 25, 50, 75, 100],
    minimal = false,
}) => {
    const navigate = useNavigate();

    const showLabel = (value) => value >= 10;

    const getLabelData = () =>
        data.sort(sortByYValue).map((item) => {
            let label = '';
            // This prevents overlap between the value label and the axis label
            if (showLabel(item.x)) {
                label = (valueFormat && valueFormat(item.x).toString()) || '';
            }
            const val = {
                x: item.x - 5,
                y: item.y,
                yOffset: minimal ? -6 : -3,
                label,
            };
            // x offset for label
            val.x -= val.label.length;
            return val;
        });

    const onValueClickHandler = (datum) => {
        if (datum.link) {
            navigate(datum.link);
        }
    };

    const getContainerProps = () => {
        const defaultContainerProps = {
            className: 'relative chart-container w-full horizontal-bar-responsive',
        };
        return merge(defaultContainerProps, containerProps);
    };

    const getPlotProps = () => {
        const sortedData = data.sort(sortByYValue);
        // This determines how far to push the bar graph to the right based on the longest axis label character's length
        const maxLength = sortedData.reduce((acc, curr) => Math.max(curr.y.length, acc), 0);

        // Magic number that makes the horizontal bars a reasonable size if there are fewer than 6
        // but shrinks them if there are 6 or more.
        const yRangeMultiplier = Math.min(41, 210 / sortedData.length);
        const yRange = [...sortedData.map((item, i) => (i + 1) * yRangeMultiplier), 0];
        const defaultPlotProps = {
            height: minimal ? 25 : 350,
            xDomain: [0, 102],
            yType: 'category',
            yRange,
            margin: minimal ? minimalMargin : { top: 33.3, left: Math.ceil(maxLength * 7.5) },
            stackBy: 'x',
        };
        return merge(defaultPlotProps, plotProps);
    };

    const getSeriesProps = () => {
        const defaultSeriesProps = {
            style: {
                height: minimal ? 15 : 20,
                rx: '2px',
                cursor: `${minimal ? '' : 'pointer'}`,
            },
            onValueMouseOver: null,
            onValueMouseOut: null,
            onValueClick: onValueClickHandler,
        };
        return merge(defaultSeriesProps, seriesProps);
    };

    const sortedData = data.sort(sortByYValue);

    // Generate y axis links
    const axisLinks = sortedData.reduce((acc, curr) => {
        if (curr.link) {
            acc[curr.y] = curr.link;
        }
        return acc;
    }, {});

    const containerPropsMerged = getContainerProps();
    const plotPropsMerged = getPlotProps();
    const seriesPropsMerged = getSeriesProps();

    function tickFormat(value) {
        let inner = value;
        if (axisLinks[value]) {
            inner = (
                <Link
                    style={{ fill: 'var(--pf-v5-global--link--Color)' }}
                    className="text-sm"
                    to={axisLinks[value]}
                >
                    {value}
                </Link>
            );
        }

        return <tspan>{inner}</tspan>;
    }

    return (
        <div {...containerPropsMerged}>
            <FlexibleWidthXYPlot {...plotPropsMerged}>
                {/* Empty area bar background */}

                {!minimal && <VerticalGridLines tickValues={tickValues} />}

                {!minimal && (
                    <XAxis
                        orientation="top"
                        tickSize={0}
                        tickFormat={valueFormat}
                        tickValues={tickValues}
                    />
                )}

                {/* Empty Background */}
                <HorizontalBarSeries
                    data={sortedData.map((item) => ({
                        x: 0,
                        x0: Math.ceil(sortedData[0].x / 5) * 5,
                        y: item.y,
                        link: item.link,
                    }))}
                    color="var(--pf-v5-global--palette--black-200)"
                    style={{
                        height: seriesPropsMerged.style.height,
                        rx: '2',
                        ry: '2',
                        cursor: `${minimal ? '' : 'pointer'}`,
                    }}
                    onValueClick={onValueClickHandler}
                />

                {/* Values */}
                <HorizontalBarSeries
                    data={sortedData.map((item) => ({ ...item, color: getColor(item.x) }))}
                    {...seriesPropsMerged}
                    colorType="literal"
                />
                <LabelSeries
                    data={getLabelData()}
                    className="text-xs pointer-events-none theme-light"
                    labelAnchorY="no-change"
                    labelAnchorX="end-alignment"
                    style={{
                        fill: '#ffffff',
                        cursor: `${minimal ? '' : 'pointer'}`,
                    }}
                />

                {!minimal && (
                    <YAxis tickSize={0} top={26} className="text-xs" tickFormat={tickFormat} />
                )}
            </FlexibleWidthXYPlot>
        </div>
    );
};

HorizontalBarChart.propTypes = {
    data: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    containerProps: PropTypes.shape({}),
    plotProps: PropTypes.shape({}),
    seriesProps: PropTypes.shape({}),
    valueFormat: PropTypes.func,
    tickValues: PropTypes.arrayOf(PropTypes.number),
    minimal: PropTypes.bool,
};

export default HorizontalBarChart;
