import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Panel from 'Components/Panel';

class NetworkPolicySimulator extends Component {
    static propTypes = {
        onClose: PropTypes.func.isRequired,
        /* eslint-disable */
        onYamlUpload: PropTypes.func.isRequired
    };

    renderSidePanel() {
        const header = 'Network Policy Simulator';
        return (
            <Panel
                className="border-r-0"
                header={header}
                onClose={this.props.onClose}
                closeButtonClassName="bg-success-500 hover:bg-success-500"
                closeButtonIconColor="text-white"
            >
                Network Policy Simulator Side panel
            </Panel>
        );
    }

    render() {
        return (
            <div className="h-full absolute pin-r pin-b w-2/5 pt-1 pb-1 pr-1">
                {this.renderSidePanel()}
            </div>
        );
    }
}

export default NetworkPolicySimulator;
