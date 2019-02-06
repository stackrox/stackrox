import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

class MultiGaugeDetailSection extends Component {
    static propTypes = {
        onClick: PropTypes.func.isRequired,
        data: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
        selectedData: PropTypes.shape({}),
        colors: PropTypes.arrayOf(PropTypes.string)
    };

    static defaultProps = {
        selectedData: null,
        colors: []
    };

    state = {
        selectedData: { ...this.props.selectedData }
    };

    componentWillReceiveProps(props) {
        this.setState({ selectedData: props.selectedData });
    }

    onClick = (data, arc, index) => () => {
        if (
            this.state.selectedData &&
            data &&
            arc === this.state.selectedData.arc &&
            data.title === this.state.selectedData.title
        ) {
            this.setState({ selectedData: null });
            this.props.onClick(null);
            return;
        }
        const selectedData = {
            ...data,
            arc,
            index
        };
        this.setState({ selectedData });
        this.props.onClick(selectedData);
    };

    getSingleGaugeContent = (d, idx) => {
        const selectedGauge = this.state.selectedData;
        const passingClassName =
            selectedGauge && selectedGauge.arc === 'inner' ? 'text-success-500' : '';
        const failingClassName =
            selectedGauge && selectedGauge.arc === 'outer' ? 'text-alert-500' : '';
        return (
            <div key={`${d.title}-${d.arc}`}>
                <div
                    className={`widget-detail-bullet ${
                        selectedGauge && selectedGauge.arc === 'inner' ? '' : 'text-base-500'
                    }`}
                >
                    <button
                        type="button"
                        className={`text-base-500 underline pl-2 cursor-pointer ${passingClassName}`}
                        onClick={this.onClick(d, 'inner', idx)}
                    >
                        {d.passing} passing controls
                    </button>
                </div>
                <div
                    key={d.title}
                    className={`widget-detail-bullet ${
                        selectedGauge && selectedGauge.arc === 'outer' ? '' : 'text-base-500'
                    }`}
                >
                    <button
                        type="button"
                        className={`text-base-500 underline pl-2 cursor-pointer ${failingClassName}`}
                        onClick={this.onClick(d, 'outer', idx)}
                    >
                        {d.failing} failing controls
                    </button>
                </div>
            </div>
        );
    };

    getMultiGaugeContent = (d, idx) => {
        const { colors } = this.props;
        const selectedGauge = this.state.selectedData;
        const passingClassName =
            selectedGauge && selectedGauge.arc === 'inner' ? 'text-success-500' : '';
        const failingClassName =
            selectedGauge && selectedGauge.arc === 'outer' ? 'text-alert-500' : '';
        const percentagePassing = Math.round((d.passing / (d.passing + d.failing)) * 100);
        const percentageFailing = Math.round((d.failing / (d.passing + d.failing)) * 100);
        return (
            <div
                key={`${d.title}-${d.arc}`}
                className={`widget-detail-bullet ${
                    selectedGauge && selectedGauge.index === idx ? '' : 'text-base-500'
                }`}
            >
                <Icon.Square fill={colors[idx]} stroke={d.color} className="h-3 w-3 pt-1" />
                <span className="pl-1">{d.title}</span>
                <button
                    type="button"
                    className={`text-base-500 underline pl-2 cursor-pointer ${selectedGauge &&
                        selectedGauge.index === idx &&
                        passingClassName}`}
                    onClick={this.onClick(d, 'inner', idx)}
                >
                    {percentagePassing}%
                </button>
                <button
                    type="button"
                    className={`text-base-500 underline pl-2 cursor-pointer ${selectedGauge &&
                        selectedGauge.index === idx &&
                        failingClassName}`}
                    onClick={this.onClick(d, 'outer', idx)}
                >
                    {percentageFailing}%
                </button>
            </div>
        );
    };

    getContent = () => (
        <div className="pt-3 pl-3">
            {this.props.data.map((d, idx) =>
                this.props.data.length === 1
                    ? this.getSingleGaugeContent(d, idx)
                    : this.getMultiGaugeContent(d, idx)
            )}
        </div>
    );

    render() {
        return (
            <div className="border-base-300 border-l flex flex-col justify-between w-full">
                {this.getContent()}
            </div>
        );
    }
}

export default MultiGaugeDetailSection;
