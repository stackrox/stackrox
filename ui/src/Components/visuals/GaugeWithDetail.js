import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import findIndex from 'lodash/findIndex';
import URLService from 'modules/URLService';
import colors from 'constants/visuals/colors';
import { CLIENT_SIDE_SEARCH_OPTIONS } from 'constants/searchOptions';

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

class GaugeWithDetail extends Component {
    static propTypes = {
        data: PropTypes.arrayOf(
            PropTypes.shape({
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
        history: ReactRouterPropTypes.history.isRequired,
        match: ReactRouterPropTypes.match.isRequired,
        location: ReactRouterPropTypes.location.isRequired
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
        this.setState({ data: this.calculateMultiGaugeData(this.props.data) });
        this.setDefaultSelectedData();
    }

    componentWillReceiveProps(nextProps) {
        if (this.state.selectedData) return;
        if (nextProps.data !== this.props.data) {
            const data = this.calculateMultiGaugeData(nextProps.data);
            this.setState({ data });
        }
        this.setDefaultSelectedData();
    }

    setDefaultSelectedData = () => {
        const { data, match, location } = this.props;
        const params = URLService.getParams(match, location);
        const complianceState = params.query[CLIENT_SIDE_SEARCH_OPTIONS.COMPLIANCE.STATE];
        const standardName = params.query.Standard;
        if (complianceState && standardName) {
            const arc = complianceState.toLowerCase() === 'passing' ? 'inner' : 'outer';
            const index = findIndex(data, datum => datum.title === standardName);
            if (index !== -1) {
                const selectedData = { ...data[index] };
                selectedData.arc = arc;
                selectedData.index = index;
                this.setSelectedData(selectedData);
            }
        }
    };

    getPropsData = () => {
        const modifiedData = this.calculateMultiGaugeData(this.props.data);
        this.setState({ data: modifiedData });
        return modifiedData;
    };

    calculateMultiGaugeData = datum => {
        if (!datum.length) return null;
        const pi = Math.PI;
        const fullAngle = 2 * pi;
        // TODO: Dynamic technique to assign  radius & font size to the gauges.
        let radius = datum.length > 1 ? 1 : 1.5;
        LABEL_STYLE.fontSize = datum.length > 1 ? '24px' : '36px';
        const data = [];
        [...datum].forEach((d, index) => {
            const { value: passingValue } = d.passing;
            const { value: failingValue } = d.failing;
            const radius0 = radius + 0.1;
            const radius1 = radius + 0.2;
            radius = radius1;
            const outerCircle = {
                ...d,
                color: colors[index],
                angle0: 2 * pi * (passingValue / (passingValue + failingValue)),
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
                angle: 2 * pi * (passingValue / (passingValue + failingValue)),
                radius0,
                radius: radius1,
                index,
                arc: 'inner'
            };
            data.push(outerCircle, innerCircle);
        });
        return data;
    };

    setPathWithLink = selectedData => {
        if (
            selectedData &&
            this.state.selectedData &&
            selectedData.arc === this.state.selectedData.arc &&
            selectedData.title === this.state.selectedData.title
        ) {
            this.props.history.replace(selectedData.defaultLink);
        }
        if (!selectedData) {
            this.props.history.replace(this.props.data[0].defaultLink);
        } else {
            this.props.history.replace(
                selectedData.arc === 'outer' ? selectedData.failing.link : selectedData.passing.link
            );
        }
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

    onMultiGaugeDetailClick = data => {
        this.setSelectedData(data);
        this.setPathWithLink(data);
    };

    onArcClick = data => {
        const { selectedData } = this.state;
        const newData = selectedData ? null : data;
        this.setSelectedData(newData);
        this.setPathWithLink(newData);
    };

    getTotalPassing = () => {
        const { data } = this.state;
        let totalPassing = 0;
        if (data) {
            const totalValues = data.reduce(
                (accumulator, d) => {
                    const { value: passingValue } = d.passing;
                    const { value: failingValue } = d.failing;
                    return {
                        passing: accumulator.passing + passingValue,
                        total: accumulator.total + passingValue + failingValue
                    };
                },
                { passing: 0, total: 0 }
            );
            if (totalValues.total === 0) return 0;
            totalPassing = Math.round((totalValues.passing / totalValues.total) * 100);
        }
        return totalPassing;
    };

    getSelectedPassingFailing = () => {
        let value = 0;
        const { arc, passing, failing } = this.state.selectedData;
        const { value: passingValue } = passing;
        const { value: failingValue } = failing;
        // 'inner' refers to passing and 'outer' refers to failing
        if (passingValue === 0 && failingValue === 0) return 0;
        value =
            arc === 'inner'
                ? Math.round((passingValue / (passingValue + failingValue)) * 100)
                : Math.round((failingValue / (passingValue + failingValue)) * 100);
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
                        y: this.props.data.length > 1 ? -0.8 : -1.3,
                        label: `${label}%`,
                        style: LABEL_STYLE
                    }
                ]}
            />
        );
    };

    getHint = () => {
        if (!this.state.hoveredCell) return null;
        const { hoveredCell } = this.state;
        const { value: passingValue } = hoveredCell.passing;
        const { value: failingValue } = hoveredCell.failing;
        return (
            <Hint value={buildValue(hoveredCell)}>
                <div className="text-base-600 text-xs p-2 pb-1 pt-1 border z-10 border-tertiary-400 bg-tertiary-200 rounded min-w-32">
                    <h1 className="text-uppercase border-b-2 border-base-400 leading-loose text-xs pb-1">
                        {hoveredCell.title}
                    </h1>
                    <div>
                        {hoveredCell.arc === 'inner' && (
                            <div className="py-2">Passing: {passingValue}</div>
                        )}
                        {hoveredCell.arc !== 'inner' && (
                            <div className="py-2">Failing: {failingValue}</div>
                        )}
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
            <div className="flex w-full">
                <XYPlot
                    xDomain={[-2, 4]}
                    yDomain={[4, 4]}
                    width={200}
                    height={200}
                    className="w-48 z-1"
                >
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
                    onClick={this.onMultiGaugeDetailClick}
                    selectedData={this.state.selectedData}
                    colors={colors}
                />
            </div>
        );
    }
}

export default withRouter(GaugeWithDetail);
