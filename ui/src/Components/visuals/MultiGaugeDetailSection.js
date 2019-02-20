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
            selectedGauge && selectedGauge.arc === 'inner' ? 'text-success-700' : '';
        const failingClassName =
            selectedGauge && selectedGauge.arc === 'outer' ? 'text-alert-700' : '';
        const { value: passingValue } = d.passing;
        const { value: failingValue } = d.failing;
        return (
            <div key={`${d.title}-${d.arc}`}>
                <div
                    className={`widget-detail-bullet ${
                        selectedGauge && selectedGauge.arc === 'inner' ? '' : 'text-base-500'
                    }`}
                >
                    <button
                        type="button"
                        className={`text-base-600 font-600 hover:text-success-700 underline cursor-pointer ${passingClassName}`}
                        onClick={this.onClick(d, 'inner', idx)}
                    >
                        {passingValue} passing controls
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
                        className={`text-base-600 font-600 hover:text-alert-700 underline cursor-pointer ${failingClassName}`}
                        onClick={this.onClick(d, 'outer', idx)}
                    >
                        {failingValue} failing controls
                    </button>
                </div>
            </div>
        );
    };

    getMultiGaugeContent = (d, idx) => {
        const { colors } = this.props;
        const selectedGauge = this.state.selectedData;
        const passingClassName =
            selectedGauge && selectedGauge.arc === 'inner' ? 'text-success-600' : '';
        const failingClassName =
            selectedGauge && selectedGauge.arc === 'outer' ? 'text-alert-600' : '';
        const { value: passingValue } = d.passing;
        const { value: failingValue } = d.failing;
        const percentagePassing = Math.round((passingValue / (passingValue + failingValue)) * 100);
        const percentageFailing = 100 - percentagePassing;
        return (
            <div
                key={`${d.title}-${d.arc}`}
                className={`widget-detail-bullet flex items-center word-break leading-tight border-b border-base-300 py-2 ${
                    selectedGauge && selectedGauge.index === idx ? '' : 'text-base-600 font-600'
                }`}
            >
                <div>
                    <Icon.Square fill={colors[idx]} stroke={d.color} className="h-2 w-2" />
                </div>
                <span className="pl-1 font-600 truncate">{d.title}</span>
                <div className="ml-auto text-right flex flex-no-shrink items-center">
                    <button
                        type="button"
                        title="Passing"
                        className={`text-sm text-base-600 font-600 hover:text-success-600 underline pl-2 cursor-pointer ${selectedGauge &&
                            selectedGauge.index === idx &&
                            passingClassName}`}
                        onClick={this.onClick(d, 'inner', idx)}
                    >
                        {percentagePassing}%
                    </button>
                    <span className="px-1"> / </span>
                    <button
                        type="button"
                        title="Failing"
                        className={`text-sm text-base-600 hover:text-alert-600 font-600 underline cursor-pointer ${selectedGauge &&
                            selectedGauge.index === idx &&
                            failingClassName}`}
                        onClick={this.onClick(d, 'outer', idx)}
                    >
                        {percentageFailing}%
                    </button>
                </div>
            </div>
        );
    };

    getContent = () => (
        <div className="pl-3">
            {this.props.data.map((d, idx) =>
                this.props.data.length === 1
                    ? this.getSingleGaugeContent(d, idx)
                    : this.getMultiGaugeContent(d, idx)
            )}
        </div>
    );

    render() {
        return (
            <div className="border-base-300 border-l flex flex-col justify-between w-full text-sm">
                {this.getContent()}
            </div>
        );
    }
}

export default MultiGaugeDetailSection;
