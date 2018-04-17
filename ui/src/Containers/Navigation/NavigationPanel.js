import React, { Component } from 'react';
import { NavLink as Link } from 'react-router-dom';
import PropTypes from 'prop-types';
import { fetchClusters } from 'services/ClustersService';

const navLinks = [
    {
        text: 'System Policies',
        to: '/main/policies'
    },
    {
        text: 'Integrations',
        to: '/main/integrations'
    }
];

class NavigationPanel extends Component {
    static propTypes = {
        panelType: PropTypes.string.isRequired,
        onClose: PropTypes.func.isRequired
    };

    constructor(props) {
        super(props);
        this.state = {
            clusters: null
        };
        this.panels = {
            configure: this.renderConfigurePanel,
            compliance: this.renderCompliancePanel
        };
    }

    componentDidMount() {
        this.getClusters();
    }

    getClusters = () =>
        fetchClusters().then(response => {
            const { clusters } = response.response;
            this.setState({ clusters });
        });

    handleKeyDown = () => {};

    renderConfigurePanel = () => (
        <ul className="flex flex-col list-reset uppercase tracking-wide bg-primary-700 border-r border-primary-800">
            <li className="border-b-2 border-primary-800 px-1 py-5 pl-2 pr-2 text-white text-base-800">
                Configure Prevent Settings
            </li>
            {navLinks.map(navLink => (
                <li key={navLink.text} className="flex flex-col text-sm">
                    <Link
                        to={navLink.to}
                        onClick={this.props.onClose(true, 'configure')}
                        className="no-underline text-white px-1 border-b py-5 border-primary-400 pl-2 pr-2 hover:bg-primary-600"
                    >
                        {navLink.text}
                    </Link>
                </li>
            ))}
        </ul>
    );

    renderCompliancePanel = () => {
        if (!this.state.clusters) return '';
        return (
            <ul className="flex flex-col list-reset uppercase tracking-wide bg-primary-700 border-r border-primary-800">
                <li className="border-b-2 border-primary-800 px-1 py-5 pl-2 pr-2 text-white text-base-800">
                    View Benchmarks per Cluster
                </li>
                {!this.state.clusters.length && (
                    <li className="flex flex-col flex-1 pl-2 pr-2 justify-center text-center text-white text-sm">
                        No clusters available
                    </li>
                )}
                {this.state.clusters.map(cluster => (
                    <li key={cluster.id} className="flex flex-col text-sm">
                        <Link
                            to={`/main/compliance/${cluster.id}`}
                            onClick={this.props.onClose(true, 'compliance')}
                            className="no-underline text-white px-1 border-b py-5 border-primary-400 pl-2 pr-2 hover:bg-primary-600"
                        >
                            {cluster.name}
                        </Link>
                    </li>
                ))}
            </ul>
        );
    };

    render() {
        return (
            <div className="navigation-panel w-full flex">
                {this.panels[this.props.panelType]()}
                <div
                    role="button"
                    tabIndex="0"
                    className="flex-1 opacity-50 bg-primary-700"
                    onClick={this.props.onClose(true)}
                    onKeyDown={this.handleKeyDown}
                />
            </div>
        );
    }
}

export default NavigationPanel;
