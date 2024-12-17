import React from 'react';
import {
    FlexibleXYPlot,
    XAxis,
    YAxis,
    VerticalGridLines,
    HorizontalGridLines,
    VerticalBarSeries,
} from 'react-vis';
import PropTypes from 'prop-types';
import { Link, useNavigate } from 'react-router-dom';
import DiscreteColorLegend from 'react-vis/dist/legends/discrete-color-legend';
import merge from 'deepmerge';

import { standardBaseTypes } from 'constants/entityTypes';

import { verticalBarColors } from './colorsForCompliance';

function VerticalClusterBar({
    id = '',
    data,
    containerProps = {},
    plotProps = {},
    seriesProps = {},
    tickValues = [25, 50, 75, 100],
    tickFormat = (x) => `${x}%`,
    labelLinks = {},
}) {
    const navigate = useNavigate();

    const getLegendData = () => {
        return Object.keys(data)
            .sort()
            .map((key, i) => ({
                title: standardBaseTypes[key] || key,
                color: verticalBarColors[i % verticalBarColors.length],
            }));
    };

    // Default props
    const defaultPlotProps = {
        xType: 'ordinal',
        yDomain: [0, 100],
        height: 270,
    };

    const defaultContainerProps = {
        className: 'relative chart-container w-full horizontal-bar-responsive',
    };

    const defaultSeriesProps = {
        // animation: true, //causes onValueMouseOut to fail https://github.com/uber/react-vis/issues/381
        barWidth: 0.5,
        style: {
            opacity: '.85',
            width: '10px',
            ry: '2px',
            cursor: 'pointer',
        },
        onValueClick: (datum) => {
            if (datum.link) {
                navigate(datum.link);
            }
        },
    };

    // Merge props
    const containerPropsMerged = merge(defaultContainerProps, containerProps);
    const plotPropsMerged = merge(defaultPlotProps, plotProps);
    const seriesPropsMerged = merge(defaultSeriesProps, seriesProps);

    function formatTicks(value) {
        let inner = value;
        if (labelLinks[value]) {
            inner = (
                <Link
                    style={{ fill: 'currentColor' }}
                    className="underline text-base-600 hover:text-primary-700"
                    to={labelLinks[value]}
                >
                    {value}
                </Link>
            );
        }

        return <tspan>{inner}</tspan>;
    }

    // Calculate unique cluster names
    let clusterNames = new Set();
    Object.keys(data).forEach((dataSetKey) => {
        const dataSet = data[dataSetKey];
        dataSet.forEach((datum) => {
            clusterNames.add(datum.x);
        });
    });
    clusterNames = Array.from(clusterNames);

    // Create Barseries for each data set
    const series = [];
    Object.keys(data)
        .sort()
        .forEach((key, i) => {
            series.push(
                <VerticalBarSeries
                    data={data[key]}
                    color={verticalBarColors[i % verticalBarColors.length]}
                    className={`vertical-cluster-bar-${standardBaseTypes[key]}`}
                    {...seriesPropsMerged}
                    key={key}
                />
            );
        });

    return (
        <div {...containerPropsMerged} data-testid={id}>
            <div className="flex flex-col h-full">
                <FlexibleXYPlot {...plotPropsMerged}>
                    <VerticalGridLines
                        left={330 / clusterNames.length / 2 + 30}
                        tickValues={clusterNames.slice(0, clusterNames.length - 1)}
                    />
                    <HorizontalGridLines tickValues={tickValues} />
                    <YAxis tickValues={tickValues} tickSize={0} tickFormat={tickFormat} />
                    {series}

                    <XAxis tickSize={0} tickFormat={formatTicks} />
                </FlexibleXYPlot>
                <div>
                    <DiscreteColorLegend
                        orientation="horizontal"
                        items={getLegendData()}
                        colors={verticalBarColors}
                        className="horizontal-bar-legend"
                    />
                </div>
            </div>
        </div>
    );
}

VerticalClusterBar.propTypes = {
    id: PropTypes.string,
    data: PropTypes.shape({}).isRequired,
    containerProps: PropTypes.shape({}),
    plotProps: PropTypes.shape({}),
    seriesProps: PropTypes.shape({}),
    tickValues: PropTypes.arrayOf(PropTypes.number),
    tickFormat: PropTypes.func,
    labelLinks: PropTypes.shape({}),
};

export default VerticalClusterBar;
