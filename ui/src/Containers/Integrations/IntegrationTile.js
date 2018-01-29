import React, { Component } from 'react';
import PropTypes from 'prop-types';

import * as Icon from 'react-feather';

class IntegrationTile extends Component {
    static propTypes = {
        integration: PropTypes.shape({
            label: PropTypes.string.isRequired,
            image: PropTypes.string.isRequired
        }).isRequired,
        onClick: PropTypes.func.isRequired,
        disabled: PropTypes.bool,
        isIntegrated: PropTypes.bool
    }

    static defaultProps = {
        disabled: false,
        isIntegrated: false
    }

    onClick = () => this.props.onClick(this.props.integration)

    renderIndicator = () => {
        if (!this.props.isIntegrated) return '';
        return <Icon.CheckSquare className="h-6 w-6 absolute pin-r pin-t m-4 text-success-500" />;
    }

    render() {
        const { integration, disabled } = this.props;
        return (
            <div className="p-3 w-1/4">
                <button
                    className={`w-full p-4 bg-white rounded-sm shadow text-center relative ${(disabled) ? 'disabled' : ''}`}
                    onClick={this.onClick}
                >
                    {this.renderIndicator()}
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
