import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

class MultiGaugeDetailSection extends Component {
    static propTypes = {
        onClick: PropTypes.func.isRequired,
        data: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
        selectedData: PropTypes.shape({
            passing: PropTypes.bool,
            failing: PropTypes.shape({
                selected: PropTypes.bool
            }),
            index: PropTypes.number
        }),
        colors: PropTypes.arrayOf(PropTypes.string)
    };

    static defaultProps = {
        selectedData: null,
        colors: []
    };

    onClick = (data, value, index) => () => {
        this.props.onClick({
            ...data,
            value,
            index
        });
    };

    getSingleGaugeContent = (d, idx) => {
        const { selectedData } = this.props;
        const passingSelected = selectedData && selectedData.passing.selected;
        const failingSelected = selectedData && selectedData.failing.selected;
        const passingClassName = passingSelected ? 'text-success-700' : '';
        const failingClassName = failingSelected ? 'text-alert-700' : '';
        const { passing, failing, skipped } = d;
        const { controls: passingValue } = passing;
        const { controls: failingValue } = failing;
        return (
            <div key={`${d.title}-${d.arc}`}>
                <div
                    data-testid="gauge-detail-bullet"
                    className={`widget-detail-bullet ${passingSelected ? '' : 'text-base-500'}`}
                >
                    <button
                        data-testid="passing-controls"
                        type="button"
                        className={`text-base-600 font-600 hover:text-success-700 underline cursor-pointer ${passingClassName}`}
                        onClick={this.onClick(d, 'passing', idx)}
                    >
                        <span data-testid="passing-controls-value">{passingValue} </span>
                        passing controls
                    </button>
                </div>
                <div
                    key={d.title}
                    data-testid="gauge-detail-bullet"
                    className={`widget-detail-bullet ${failingSelected ? '' : 'text-base-500'}`}
                >
                    <button
                        data-testid="failing-controls"
                        type="button"
                        className={`text-base-600 font-600 hover:text-alert-700 underline cursor-pointer ${failingClassName}`}
                        onClick={this.onClick(d, 'failing', idx)}
                    >
                        <span data-testid="failing-controls-value">{failingValue} </span>
                        failing controls
                    </button>
                </div>
                {skipped > 0 && (
                    <div data-testid="gauge-detail-bullet" className="widget-detail-bullet">
                        <button
                            data-testid="skipped-controls"
                            type="button"
                            className="text-base-600 font-600 pointer-events-none"
                        >
                            <span data-testid="skipped-controls-value">{skipped} </span>
                            skipped controls
                        </button>
                    </div>
                )}
            </div>
        );
    };

    getMultiGaugeContent = (d, idx) => {
        const { colors } = this.props;
        const { selectedData } = this.props;
        const passingSelected = selectedData && selectedData.passing.selected;
        const failingSelected = selectedData && selectedData.failing.selected;
        const passingClassName = passingSelected ? 'text-success-600' : '';
        const failingClassName = failingSelected ? 'text-alert-600' : '';
        const { value: passingValue } = d.passing;
        const { value: failingValue } = d.failing;
        return (
            <div
                key={`${d.title}-${d.arc}`}
                className={`widget-detail-bullet flex items-center word-break leading-tight border-b border-base-300 py-1 ${
                    selectedData && selectedData.index === idx ? '' : 'text-base-600 font-600'
                }`}
            >
                <div>
                    <Icon.Square fill={colors[idx]} stroke={d.color} className="h-2 w-2" />
                </div>
                <span className="pl-1 font-600 truncate">{d.title}</span>
                <div className="ml-auto text-right flex flex-shrink-0 items-center">
                    <button
                        type="button"
                        title="Passing"
                        className={`text-sm text-base-600 font-600 hover:text-success-600 underline pl-2 cursor-pointer ${
                            selectedData === d ? passingClassName : ''
                        }`}
                        onClick={this.onClick(d, 'passing', idx)}
                    >
                        {passingValue} Passing
                    </button>
                    <span className="px-1"> / </span>
                    <button
                        type="button"
                        title="Failing"
                        className={`text-sm text-base-600 hover:text-alert-600 font-600 underline cursor-pointer ${
                            selectedData === d ? failingClassName : ''
                        }`}
                        onClick={this.onClick(d, 'failing', idx)}
                    >
                        {failingValue} Failing
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
