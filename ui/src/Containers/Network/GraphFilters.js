import React, { Component } from 'react';
import PropTypes from 'prop-types';

import { ACTIVE_STATE, ALLOWED_STATE, ALL_STATE } from 'constants/networkGraph';

const baseButtonClassName =
    'flex-no-shrink px-2 py-px border-2 rounded-sm uppercase text-xs font-700';
const buttonClassName = `${baseButtonClassName} border-base-400 hover:bg-primary-200 text-base-600`;
const activeButtonClassName = `${baseButtonClassName} bg-primary-300 border-primary-400 hover:bg-primary-200 text-primary-700 border-l-2 border-r-2`;

class GraphFilters extends Component {
    static propTypes = {
        onFilter: PropTypes.func.isRequired,
        offset: PropTypes.bool.isRequired
    };

    constructor(props, context) {
        super(props, context);

        this.state = {
            value: ALL_STATE
        };
    }

    handleChange = value => () => {
        this.setState({ value });
        this.props.onFilter(value);
    };

    render() {
        const { value } = this.state;
        return (
            <div
                className={`absolute pin-t pin-l px-2 py-2 ${
                    this.props.offset ? 'mt-8' : 'mt-2'
                } ml-2 absolute z-1 bg-primary-100 uppercase flex items-center text-sm border-base-400 border-2`}
            >
                <span className="text-base-500 font-700 mr-2">Connections:</span>
                <div className="flex items-center">
                    <button
                        type="button"
                        value={value}
                        className={`${
                            value === ACTIVE_STATE ? activeButtonClassName : buttonClassName
                        }
                        ${value === ALLOWED_STATE && 'border-r-0'}`}
                        onClick={this.handleChange(ACTIVE_STATE)}
                    >
                        Active
                    </button>
                    <button
                        type="button"
                        value={value}
                        className={`${
                            value === ALLOWED_STATE
                                ? activeButtonClassName
                                : `${buttonClassName} border-l-0 border-r-0`
                        }`}
                        onClick={this.handleChange(ALLOWED_STATE)}
                    >
                        Allowed
                    </button>
                    <button
                        type="button"
                        value={value}
                        className={`${
                            value === ALL_STATE ? activeButtonClassName : buttonClassName
                        } 
                        ${value === ALLOWED_STATE && 'border-l-0'}`}
                        onClick={this.handleChange(ALL_STATE)}
                    >
                        All
                    </button>
                </div>
            </div>
        );
    }
}

export default GraphFilters;
