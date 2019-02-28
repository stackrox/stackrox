import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Message from 'Components/Message';
import * as Icon from 'react-feather';

class ErrorView extends Component {
    static propTypes = {
        yamlFile: PropTypes.shape({
            name: PropTypes.string.isRequired,
            content: PropTypes.string.isRequired
        }).isRequired,
        errorMessage: PropTypes.string.isRequired,
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
            <Icon.Maximize2 className="h-4 w-4 text-base-500 hover:text-base-600" />
        ) : (
            <Icon.Minimize2 className="h-4 w-4 text-base-500 hover:text-base-600" />
        );
        return (
            <button
                type="button"
                className="absolute pin-r pin-t h-10 w-10 border-base-200 z-10"
                onClick={this.toggleCollapse}
            >
                {icon}
            </button>
        );
    };

    renderYamlFile = () => {
        const { name, content } = this.props.yamlFile;
        return (
            <div className="flex flex-1 flex-col bg-base-100 relative h-full">
                <div className="border-b border-base-300 p-3 text-base-600 font-700">{name}</div>
                {this.renderCollapseButton()}
                <div className="overflow-auto p-3">
                    <pre className="leading-tight whitespace-pre-wrap word-break">{content}</pre>
                </div>
            </div>
        );
    };

    render() {
        return (
            <section className="bg-base-100 flex flex-col shadow text-base-600 border border-base-200 m-3 mt-4 mb-4 overflow-hidden h-full">
                <Message type="error" message={this.props.errorMessage} />
                {this.renderYamlFile()}
            </section>
        );
    }
}

export default ErrorView;
