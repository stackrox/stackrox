import React, { Component } from 'react';
import PropTypes from 'prop-types';

class IntegrationTile extends Component {
    static propTypes = {
        integration: PropTypes.shape({
            label: PropTypes.string.isRequired,
            image: PropTypes.string.isRequired
        }).isRequired,
        onClick: PropTypes.func.isRequired,
        disabled: PropTypes.bool,
        numIntegrations: PropTypes.number
    };

    static defaultProps = {
        disabled: false,
        numIntegrations: 0
    };

    onClick = () => this.props.onClick(this.props.integration);

    renderIndicator = () => {
        if (this.props.numIntegrations === 0) return null;
        return (
            <span className="flex h-6 absolute pin-r pin-t m-2 p-2 items-center justify-center text-success-500 font-bold text-xl border-2 border-success-300">
                {this.props.numIntegrations}
            </span>
        );
    };

    render() {
        const { integration, disabled } = this.props;
        return (
            <div className="p-3 w-1/4">
                <button
                    className={`w-full p-4 bg-white rounded-sm shadow text-center relative ${
                        disabled ? 'disabled' : ''
                    } ${this.props.numIntegrations !== 0 && 'border-2 border-success-400'}`}
                    onClick={this.onClick}
                >
                    {this.renderIndicator()}
                    <img
                        className="w-24 h-24 mb-4"
                        src={integration.image}
                        alt={integration.label}
                    />
                    <div className="font-bold text-xl pt-4  border-t border-base-200">
                        {integration.label}
                    </div>
                </button>
            </div>
        );
    }
}

export default IntegrationTile;
