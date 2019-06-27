import React, { useState } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import colors from 'constants/visuals/colors';
import { XYPlot, ArcSeries, LabelSeries, Hint } from 'react-vis';
import MultiGaugeDetailSection from './MultiGaugeDetailSection';

const LABEL_STYLE = {
    textAnchor: 'middle',
    fill: 'var(--primary-800)',
    fontWeight: 400
};

const buildValue = hoveredCell => {
    const { radius, angle, angle0 } = hoveredCell;
    const truedAngle = (angle + angle0) / 2;
    return {
        x: radius * Math.cos(truedAngle),
        y: radius * Math.sin(truedAngle)
    };
};

const GaugeWithDetail = ({ data, history }) => {
    const [hoveredCell, setHoveredCell] = useState();

    const selectedData = data.find(datum => {
        return datum.passing.selected || datum.failing.selected;
    });

    const failingSelected = selectedData && selectedData.failing.selected;
    const passingSelected = selectedData && selectedData.passing.selected;
    const totalPassing = data.reduce((acc, datum) => acc + datum.passing.value, 0);
    const totalFailing = data.reduce((acc, datum) => acc + datum.failing.value, 0);
    const totalChecks = totalPassing + totalFailing;
    const pctPassing = totalChecks ? Math.round((totalPassing / totalChecks) * 100) : 0;
    const pctFailing = totalChecks ? Math.round((totalFailing / totalChecks) * 100) : 0;

    function calculateGaugeData(inputData) {
        if (!inputData.length) return null;
        const pi = Math.PI;
        const fullAngle = 2 * pi;
        // TODO: Dynamic technique to assign  radius & font size to the gauges.
        let radius = inputData.length > 1 ? 1 : 1.5;
        LABEL_STYLE.fontSize = inputData.length > 1 ? '24px' : '36px';
        const returnData = [];

        [...inputData].forEach((d, index) => {
            const { value: passingValue } = d.passing;
            const { value: failingValue } = d.failing;
            const radius0 = radius + 0.1;
            const radius1 = radius + 0.2;
            radius = radius1;
            const failingCircle = {
                ...d,
                color: failingSelected ? 'var(--alert-400)' : colors[index],
                angle0: 2 * pi * (passingValue / (passingValue + failingValue)),
                angle: fullAngle,
                opacity: failingSelected ? 1 : 0.2,
                radius0,
                radius: radius1,
                index,
                arc: 'outer'
            };

            const passingCircle = {
                ...d,
                color: passingSelected ? 'var(--success-400)' : colors[index],
                angle0: 0,
                angle: 2 * pi * (passingValue / (passingValue + failingValue)),
                radius0,
                radius: radius1,
                index,
                arc: 'inner',
                opacity: failingSelected ? 0.2 : 1
            };
            returnData.push(failingCircle, passingCircle);
        });
        return returnData;
    }
    const gaugeData = calculateGaugeData(data);

    function onClick(d) {
        history.replace(d.value === 'failing' ? d.failing.link : d.passing.link);
    }

    function getHint() {
        if (!hoveredCell) return null;
        const { passing, failing, title } = hoveredCell;
        const { value: passingValue, controls: passingControls } = passing;
        const { value: failingValue, controls: failingControls } = failing;
        const totalValue = passingValue + failingValue;
        const totalControls = passingControls + failingControls;
        const passingPercentage = Math.round((passingValue / totalValue) * 100);
        const failingPercentage = Math.round((failingValue / totalValue) * 100);
        return (
            <Hint value={buildValue(hoveredCell)}>
                <div className="text-base-600 text-xs p-2 pb-1 pt-1 border z-10 border-tertiary-400 bg-tertiary-200 rounded min-w-32">
                    <h1 className="text-uppercase border-b-2 border-base-400 leading-loose text-xs pb-1">
                        {title}
                    </h1>
                    <div>
                        {hoveredCell.arc === 'inner' && (
                            <div className="py-2">
                                {passingPercentage}% of checks passing across {totalControls}{' '}
                                controls
                            </div>
                        )}
                        {hoveredCell.arc !== 'inner' && (
                            <div className="py-2">
                                {failingPercentage}% of checks failing across {totalControls}{' '}
                                controls
                            </div>
                        )}
                    </div>
                </div>
            </Hint>
        );
    }

    function onValueMouseOver(d) {
        setHoveredCell(d);
    }
    function onValueMouseOut() {
        setHoveredCell();
    }

    return (
        <div className="flex w-full">
            <XYPlot
                xDomain={[-2, 4]}
                yDomain={[4, 4]}
                width={200}
                height={200}
                className="w-48 z-1"
            >
                <LabelSeries
                    data={[
                        {
                            x: 0.1,
                            y: data.length > 1 ? -0.8 : -1.3,
                            label: `${failingSelected ? pctFailing : pctPassing}%`,
                            style: LABEL_STYLE
                        }
                    ]}
                />
                {getHint()}
                <ArcSeries
                    arcClassName="cursor-pointer"
                    radiusDomain={[0, 2]}
                    data={gaugeData}
                    colorType="literal"
                    onValueClick={onClick}
                    onValueMouseOver={onValueMouseOver}
                    onValueMouseOut={onValueMouseOut}
                />
            </XYPlot>
            <MultiGaugeDetailSection
                data={data}
                onClick={onClick}
                selectedData={selectedData}
                colors={colors}
            />
        </div>
    );
};

GaugeWithDetail.propTypes = {
    data: PropTypes.arrayOf(
        PropTypes.shape({
            id: PropTypes.string.isRequired,
            title: PropTypes.string.isRequired,
            passing: PropTypes.shape({
                value: PropTypes.number.isRequired,
                link: PropTypes.string.isRequired
            }),
            failing: PropTypes.shape({
                value: PropTypes.number.isRequired,
                link: PropTypes.string.isRequired
            }),
            defaultLink: PropTypes.string.isRequired
        })
    ).isRequired,
    history: ReactRouterPropTypes.history.isRequired
};

export default withRouter(GaugeWithDetail);
