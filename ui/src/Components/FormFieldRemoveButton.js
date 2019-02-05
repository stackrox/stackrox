import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

class FormFieldRemoveButton extends Component {
    static propTypes = {
        field: PropTypes.string.isRequired,
        onClick: PropTypes.func.isRequired
    };

    handleClick = () => this.props.onClick(this.props.field);

    render() {
        return (
            <div className="flex">
                <button
                    className="ml-2 p-1 px-3 rounded-r-sm text-base-100 uppercase text-center text-alert-700 hover:text-alert-800 bg-alert-200 hover:bg-alert-300 border-2 border-alert-300 flex items-center rounded"
                    onClick={this.handleClick}
                    type="button"
                >
                    <Icon.X size="20" />
                </button>
            </div>
        );
    }
}

export default FormFieldRemoveButton;
