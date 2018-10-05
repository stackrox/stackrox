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
                    className="p-1 px-3 rounded-r-sm text-base-100 uppercase text-center bg-alert-300 hover:bg-alert-400"
                    onClick={this.handleClick}
                    type="button"
                >
                    <Icon.X className="h-4 w-4" />
                </button>
            </div>
        );
    }
}

export default FormFieldRemoveButton;
