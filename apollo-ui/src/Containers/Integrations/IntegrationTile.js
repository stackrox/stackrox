import React, { Component } from 'react';
import PropTypes from 'prop-types';

class IntegrationTile extends Component {
    static propTypes = {
        integration: PropTypes.shape({
            label: PropTypes.string.isRequired,
            image: PropTypes.string.isRequired,
            disabled: PropTypes.bool
        }).isRequired,
        onClick: PropTypes.func.isRequired
    }

    onClick = () => this.props.onClick(this.props.integration)

    render() {
        const { integration } = this.props;
        return (
            <div className="p-3 w-1/4">
                <button
                    className={`w-full p-4 bg-white rounded-sm shadow text-center ${(integration.disabled) ? 'disabled' : ''}`}
                    onClick={this.onClick}
                >
                    <img className="w-24 h-24 mb-4" src={integration.image} alt={integration.label} />
                    <div className="font-bold text-xl pt-4  border-t border-base-200">
                        {integration.label}
                    </div>
                </button>
            </div>
        );
    }
}

export default IntegrationTile;
