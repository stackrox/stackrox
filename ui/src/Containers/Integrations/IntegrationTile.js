import React, { Component } from 'react';
import PropTypes from 'prop-types';

class IntegrationTile extends Component {
    static propTypes = {
        integration: PropTypes.shape({
            label: PropTypes.string.isRequired,
            image: PropTypes.string.isRequired,
            categories: PropTypes.string
        }).isRequired,
        onClick: PropTypes.func.isRequired,
        numIntegrations: PropTypes.number
    };

    static defaultProps = {
        numIntegrations: 0
    };

    onClick = () => this.props.onClick(this.props.integration);

    handleKeyUp = e => (e.key === 'Enter' ? this.props.onClick(this.props.integration) : null);

    renderIndicator = () => {
        if (this.props.numIntegrations === 0) return null;
        return (
            <span className="flex h-6 absolute pin-r pin-t m-2 p-2 items-center justify-center text-success-600 font-700 text-xl border-2 border-success-500">
                {this.props.numIntegrations}
            </span>
        );
    };

    render() {
        const { integration, numIntegrations } = this.props;
        return (
            <div className="p-3 w-full md:w-1/2 lg:w-1/3 xl:w-1/4 min-h-55">
                <div
                    className={`flex flex-col justify-between cursor-pointer border-3 border-base-100 hover:shadow-lg items-center h-full w-full bg-base-100 rounded-sm shadow text-center relative 
                    ${numIntegrations !== 0 && 'border-2 border-success-500'}`}
                    onClick={this.onClick}
                    onKeyUp={this.handleKeyUp}
                    role="button"
                    tabIndex="0"
                >
                    {this.renderIndicator()}
                    <div className="flex h-full w-full flex-col justify-center">
                        <img
                            className="w-full px-7"
                            src={integration.image}
                            alt={integration.label}
                        />
                    </div>
                    <div className="bg-tertiary-200 flex flex-col items-center justify-center min-h-16 w-full">
                        <div className="leading-loose text-2xl text-tertiary-800">
                            {integration.label}
                        </div>
                        {integration.categories !== '' &&
                            integration.categories !== undefined && (
                                <div className="font-700 text-tertiary-700 text-xs tracking-widest uppercase mb-1">
                                    {integration.categories}
                                </div>
                            )}
                    </div>
                </div>
            </div>
        );
    }
}

export default IntegrationTile;
