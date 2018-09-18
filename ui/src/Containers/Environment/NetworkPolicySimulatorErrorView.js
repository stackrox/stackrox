import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Message from 'Components/Message';

class NetworkPolicySimulatorErrorView extends Component {
    static propTypes = {
        yamlFile: PropTypes.shape({
            name: PropTypes.string.isRequired,
            content: PropTypes.string.isRequired
        }).isRequired,
        errorMessage: PropTypes.string.isRequired
    };

    renderYamlFile = () => {
        const { name, content } = this.props.yamlFile;
        return (
            <div className="flex flex-1 flex-col bg-white">
                <div className="border-b border-base-200 p-3 text-danger-600">{name}</div>
                <pre className="h-full p-3 leading-loose">{content}</pre>
            </div>
        );
    };

    render() {
        return (
            <section className="bg-white shadow text-base-600 border border-base-200 m-3">
                <Message type="error" message={this.props.errorMessage} />
                {this.renderYamlFile()}
            </section>
        );
    }
}

export default NetworkPolicySimulatorErrorView;
