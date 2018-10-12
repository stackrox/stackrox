import React, { Component } from 'react';
import * as Icon from 'react-feather';

import LegendTile from 'Components/LegendTile';

class NetworkGraphLegend extends Component {
    state = {
        isOpen: true
    };

    handleKeyUp = e => (e.key === 'Enter' ? this.toggleLegend() : null);

    toggleLegend = () => {
        const { isOpen } = this.state;
        this.setState({ isOpen: !isOpen });
    };

    renderLegendButton = () => {
        if (this.state.isOpen) return null;
        return (
            <div
                role="button"
                className="uppercase p-2 hover:bg-base-200 hover:text-primary-700 cursor-pointer"
                onClick={this.toggleLegend}
                onKeyUp={this.handleKeyUp}
                tabIndex="0"
            >
                Legend
            </div>
        );
    };

    renderLegendContent = () => {
        if (!this.state.isOpen) return null;
        return (
            <React.Fragment>
                <div className="flex justify-between border-b border-base-400 p-2 uppercase items-center">
                    Legend
                    <button type="button" className="flex" onClick={this.toggleLegend}>
                        <Icon.X className="h-3 w-3" />
                    </button>
                </div>
                <div className="bg-primary-100">
                    <div className="flex border-b border-base-400">
                        <LegendTile svgName="deployment" tooltip="Deployment" />
                        <LegendTile
                            svgName="deployment-allowed-connections"
                            tooltip="Deployment with allowed external connections"
                        />
                    </div>
                    <div className="flex border-b border-base-400">
                        <LegendTile svgName="namespace" tooltip="Namespace" />
                        <LegendTile
                            svgName="namespace-allowed-connection"
                            tooltip="Namespace with allowed external connections"
                        />
                    </div>
                    <div className="flex border-b border-base-400">
                        <LegendTile svgName="namespace-connection" tooltip="Namespace connection" />
                        <LegendTile svgName="active-connection" tooltip="Active connection" />
                        <LegendTile svgName="allowed-connection" tooltip="Allowed connection" />
                    </div>
                    <div className="flex">
                        <LegendTile
                            svgName="namespace-egress"
                            tooltip="Namespace external egress traffic"
                        />
                        <LegendTile
                            svgName="namespace-ingress"
                            tooltip="Namespace external ingress traffic"
                        />
                        <LegendTile
                            svgName="namespace-egress-ingress"
                            tooltip="Namespace external egress/ingress traffic"
                        />
                    </div>
                </div>
            </React.Fragment>
        );
    };

    render() {
        return (
            <div
                data-test-id="legend"
                className="env-graph-legend absolute pin-b pin-l mb-2 ml-2 bg-base-100 text-base-500 text-sm font-700 border-base-400 border-2 rounded-sm z-10"
            >
                {this.renderLegendButton()}
                {this.renderLegendContent()}
            </div>
        );
    }
}

export default NetworkGraphLegend;
