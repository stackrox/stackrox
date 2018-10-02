import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Message from 'Components/Message';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import * as Icon from 'react-feather';

const successMessage = 'YAML file uploaded successfully';
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
        this.setState(prevState => ({ isCollapsed: !prevState.isCollapsed }));
    };

    renderCollapseButton = () => {
        const icon = this.state.isCollapsed ? (
            <Icon.Maximize2 className="h-4 w-4 text-base-500" />
        ) : (
            <Icon.Minimize2 className="h-4 w-4 text-base-500" />
        );
        return (
            <button
                type="button"
                className="absolute pin-r pin-t h-12 w-12 border-base-200 z-10"
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
                    <div className="network-policy-success-yaml flex flex-col bg-base-100 overflow-auto">
                        <pre className="p-3 leading-loose whitespace-pre-wrap word-break">
                            {yamlFile.content}
                        </pre>
                    </div>
                </TabContent>
            </Tabs>
        );
    };

    render() {
        return (
            <section className="bg-base-100 shadow text-base-600 border border-base-200 m-3 overflow-hidden h-full">
                <Message type="info" message={successMessage} />
                <div className="relative h-full">
                    {this.renderTabs()}
                    {this.renderCollapseButton()}
                </div>
            </section>
        );
    }
}

export default NetworkPolicySimulatorSuccessView;
