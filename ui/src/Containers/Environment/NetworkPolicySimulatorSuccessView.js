import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import * as Icon from 'react-feather';

class NetworkPolicySimulatorSuccessView extends Component {
    static propTypes = {
        yamlFile: PropTypes.shape({
            name: PropTypes.string.isRequired,
            content: PropTypes.string.isRequired
        }).isRequired,
        onCollapse: PropTypes.func.isRequired
    };

    state = {
        isCollapsed: true
    };

    toggleCollapse = () => {
        this.props.onCollapse(!this.state.isCollapsed);
        this.setState({ isCollapsed: !this.state.isCollapsed });
    };

    renderCollapseButton = () => {
        const icon = this.state.isCollapsed ? (
            <Icon.Maximize2 className="h-4 w-4 text-base-500" />
        ) : (
            <Icon.Minimize2 className="h-4 w-4 text-base-500" />
        );
        return (
            <button
                className="absolute pin-r pin-t h-12 w-12 border-l border-base-200"
                onClick={this.toggleCollapse}
            >
                {icon}
            </button>
        );
    };

    renderTabs = () => {
        const { yamlFile } = this.props;
        const tabs = [{ text: yamlFile.name }];
        return (
            <Tabs headers={tabs}>
                <TabContent>
                    <div className="flex flex-1 flex-col bg-white">
                        <pre className="h-full p-3 leading-loose">{yamlFile.content}</pre>
                    </div>
                </TabContent>
            </Tabs>
        );
    };

    render() {
        return (
            <section className="bg-white shadow text-base-600 border border-base-200 m-3 relative">
                {this.renderTabs()}
                {this.renderCollapseButton()}
            </section>
        );
    }
}

export default NetworkPolicySimulatorSuccessView;
