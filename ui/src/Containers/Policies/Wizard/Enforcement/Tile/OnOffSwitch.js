import React, { Component } from 'react';
import PropTypes from 'prop-types';

class OnOffSwitch extends Component {
    static propTypes = {
        enabled: PropTypes.bool.isRequired,
        applied: PropTypes.bool.isRequired,
        onClick: PropTypes.func.isRequired
    };

    renderSwitch = () => {
        const onColor = 'border-success-300 text-success-600';
        const offColor = 'border-alert-300 text-alert-600';
        const neutralColor = 'border-primary-300 text-primary-600';

        const onSwitchColor = this.props.enabled && this.props.applied ? onColor : neutralColor;
        const offSwitchColor = this.props.enabled && !this.props.applied ? offColor : neutralColor;

        const onSwitchClass = `px-2 py-1 border-2 bg-base-100 ${onSwitchColor} font-700 rounded-sm text-base-600 text-xs uppercase`;
        const offSwitchClass = `px-2 py-1 border-2 bg-base-100 ${offSwitchColor} font-700 rounded-sm text-base-600 text-xs uppercase`;

        return (
            <div className="flex py-2 w-full justify-center">
                <button
                    type="button"
                    className={onSwitchClass}
                    onClick={this.props.onClick}
                    disabled={!this.props.enabled}
                >
                    On
                </button>
                <button
                    type="button"
                    className={offSwitchClass}
                    onClick={this.props.onClick}
                    disabled={!this.props.enabled}
                >
                    Off
                </button>
            </div>
        );
    };

    render() {
        return <div className="flex pin-b"> {this.renderSwitch()} </div>;
    }
}

export default OnOffSwitch;
