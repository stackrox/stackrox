import React, { Component } from 'react';

import { XYPlot, ArcSeries, LabelSeries, Hint } from 'react-vis';
import PropTypes from 'prop-types';
import MultiGaugeDetailSection from './MultiGaugeDetailSection';

const LABEL_STYLE = {
    textAnchor: 'middle',
    fill: 'var(--primary-800)',
    fontWeight: 400
};

const colors = [
    'var(--primary-400)',
    'var(--secondary-400)',
    'var(--tertiary-400)',
    'var(--accent-400)'
];

const buildValue = hoveredCell => {
    const { radius, angle, angle0 } = hoveredCell;
    const truedAngle = (angle + angle0) / 2;
    return {
        x: radius * Math.cos(truedAngle),
        y: radius * Math.sin(truedAngle)
    };
};

class GaugeWithDetail extends Component {
    static propTypes = {
        data: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
        dataProperty: PropTypes.string.isRequired
    };

    constructor(props) {
        super(props);
        this.state = {
            selectedData: null,
            data: null,
            hoveredCell: null
        };
    }

    componentDidMount() {
        const { dataProperty } = this.props;
        this.setState({ data: this.calculateMultiGaugeData(dataProperty) });
    }

    getPropsData = () => {
        const { dataProperty } = this.props;
        const modifiedData = this.calculateMultiGaugeData(dataProperty);
        this.setState({ data: modifiedData });
        return modifiedData;
    };

    calculateMultiGaugeData = property => {
        if (!this.props.data.length) return null;
        const pi = Math.PI;
        const fullAngle = 2 * pi;
        // TODO: Dynamic technique to assign  radius & font size to the gauges.
        let radius = this.props.data.length > 1 ? 1 : 1.5;
        LABEL_STYLE.fontSize = this.props.data.length > 1 ? '36px' : '48px';
        const data = [];
        [...this.props.data].forEach((d, index) => {
            const radius0 = radius + 0.1;
            const radius1 = radius + 0.2;
            radius = radius1;
            const outerCircle = {
                ...d,
                color: colors[index],
                angle0: 2 * pi * (d[property] / 100),
                angle: fullAngle,
                opacity: 0.2,
                radius0,
                radius: radius1,
                index,
                arc: 'outer'
            };

            const innerCircle = {
                ...d,
                color: colors[index],
                angle0: 0,
                angle: 2 * pi * (d[property] / 100),
                radius0,
                radius: radius1,
                index,
                arc: 'inner'
            };
            data.push(outerCircle, innerCircle);
        });
        return data;
    };

    setSelectedData = selectedData => {
        if (
            selectedData &&
            this.state.selectedData &&
            selectedData.arc === this.state.selectedData.arc &&
            selectedData.title === this.state.selectedData.title
        ) {
            this.setState({
                selectedData: null,
                data: this.getPropsData()
            });
            return;
        }
        const index = selectedData && selectedData.index;
        this.setState({ selectedData });
        const data = this.getPropsData();
        const newData = data.filter((val, idx) => {
            if (selectedData) {
                if (idx !== index * 2 && idx !== index * 2 + 1) {
                    Object.assign(val, { color: 'var(--base-300)', opacity: 1 });
                } else {
                    if (selectedData.arc === val.arc) {
                        Object.assign(val, { opacity: 1 });
                    } else {
                        Object.assign(val, { opacity: 0.2 });
                    }
                    if (idx === index * 2 + 1) {
                        Object.assign(val, { color: 'var(--success-400)' });
                    }

                    if (idx === index * 2) {
                        Object.assign(val, { color: 'var(--alert-400)' });
                    }
                }
            }
            return val;
        });
        this.setState({ data: newData });
    };

    onArcClick = data => this.setSelectedData(data);

    getTotalPassing = () => {
        const { data } = this.state;
        let totalPassing = 0;
        if (data) {
            const totalValues = data.reduce(
                (accumulator, value) => ({
                    passing: accumulator.passing + value.passing,
                    total: accumulator.total + value.failing + value.passing
                }),
                { passing: 0, total: 0 }
            );
            totalPassing = Math.round((totalValues.passing / totalValues.total) * 100);
        }
        return totalPassing;
    };

    getSelectedPassingFailing = () => {
        let value = 0;
        const { arc, passing, failing } = this.state.selectedData;
        // 'inner' refers to passing and 'outer' refers to failing
        value = arc === 'inner' ? passing : failing;
        return value;
    };

    getCenterLabel = () => {
        let label = '';
        const { selectedData } = this.state;
        label = selectedData ? this.getSelectedPassingFailing() : this.getTotalPassing();
        return (
            <LabelSeries
                data={[
                    {
                        x: 0.1,
                        y: this.props.data.length > 1 ? -0.85 : 1.1,
                        label: `${label}%`,
                        style: LABEL_STYLE
                    }
                ]}
            />
        );
    };

    getHint = () => {
        if (!this.state.hoveredCell) return null;
        return (
            <Hint value={buildValue(this.state.hoveredCell)}>
                <div className="text-base-600 text-xs p-2 pb-1 pt-1 border border-tertiary-400 bg-tertiary-200 rounded min-w-32">
                    <h1 className="text-uppercase border-b-2 border-base-400 leading-loose text-xs pb-1">
                        {this.state.hoveredCell.title}
                    </h1>
                    <div>
                        <div className="pt-2">Passing: {this.state.hoveredCell.passing}</div>
                        <div className="py-2">Failing: {this.state.hoveredCell.failing}</div>
                    </div>
                </div>
            </Hint>
        );
    };

    onValueMouseOver = data => this.setState({ hoveredCell: data });

    onValueMouseOut = () => this.setState({ hoveredCell: null });

    render() {
        const { data } = this.props;
        return (
            <div className="flex flex-row">
                <XYPlot xDomain={[-3, 4]} yDomain={[-3, 4]} width={275} height={250}>
                    {this.getCenterLabel()}
                    {this.getHint()}
                    <ArcSeries
                        arcClassName="cursor-pointer"
                        radiusDomain={[0, 2]}
                        data={this.state.data}
                        colorType="literal"
                        onValueClick={this.onArcClick}
                        onValueMouseOver={this.onValueMouseOver}
                        onValueMouseOut={this.onValueMouseOut}
                    />
                </XYPlot>
                <MultiGaugeDetailSection
                    data={data}
                    onClick={this.setSelectedData}
                    selectedData={this.state.selectedData}
                    colors={colors}
                />
            </div>
        );
    }
}

export default GaugeWithDetail;
