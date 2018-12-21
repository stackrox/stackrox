import React, { Component } from 'react';
import { NavLink as Link, withRouter } from 'react-router-dom';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as clusterActions } from 'reducers/clusters';

const navLinks = [
    {
        text: 'System Policies',
        to: '/main/policies'
    },
    {
        text: 'Integrations',
        to: '/main/integrations'
    },
    {
        text: 'Access Control',
        to: '/main/access'
    }
];

class NavigationPanel extends Component {
    static propTypes = {
        clusters: PropTypes.arrayOf(PropTypes.object).isRequired,
        panelType: PropTypes.string.isRequired,
        onClose: PropTypes.func.isRequired,
        selectedClusterId: PropTypes.string,
        fetchClusters: PropTypes.func.isRequired
    };

    static defaultProps = {
        selectedClusterId: ''
    };

    constructor(props) {
        super(props);
        this.panels = {
            configure: this.renderConfigurePanel,
            compliance: this.renderCompliancePanel
        };
    }

    componentDidMount() {
        this.props.fetchClusters();
    }

    isSelectedCluster = clusterId => clusterId === this.props.selectedClusterId;

    handleKeyDown = () => {};

    renderConfigurePanel = () => (
        <ul className="flex flex-col overflow-auto list-reset uppercase tracking-wide bg-primary-800 border-r border-l border-primary-900">
            <li className="border-b-2 border-primary-500 px-1 py-5 pl-2 pr-2 text-base-100 font-700">
                Configure StackRox Settings
            </li>
            {navLinks.map(navLink => (
                <li key={navLink.text} className="text-sm">
                    <Link
                        to={navLink.to}
                        onClick={this.props.onClose(true, 'configure')}
                        className="block no-underline text-base-100 px-1 font-700 border-b py-5 border-primary-900 pl-2 pr-2 hover:bg-base-700"
                    >
                        {navLink.text}
                    </Link>
                </li>
            ))}
        </ul>
    );

    renderCompliancePanel = () => {
        if (!this.props.clusters) return '';
        return (
            <ul className="flex flex-col overflow-auto list-reset uppercase tracking-wide bg-primary-800 border-r border-l border-base-900">
                <li className="border-b-2 border-primary-500 px-1 py-5 pl-2 pr-2 text-base-100 font-700">
                    View Benchmarks per Cluster
                </li>
                {!this.props.clusters.length && (
                    <li className="flex flex-col flex-1 pl-2 pr-2 justify-center text-center text-base-100 text-sm">
                        No clusters available
                    </li>
                )}
                {this.props.clusters.map(cluster => (
                    <li key={cluster.id} className="text-sm">
                        <Link
                            to={`/main/compliance/${cluster.id}`}
                            onClick={this.props.onClose(true, 'compliance')}
                            className={`block no-underline text-base-100 px-1 border-b font-700 py-5 border-primary-900 pl-2 pr-2 hover:bg-base-700 ${
                                this.isSelectedCluster(cluster.id)
                                    ? 'bg-primary-700 hover:bg-primary-700'
                                    : ''
                            }`}
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

const getSelectedClusterId = (state, props) => props.location.pathname.split('/').pop();

const mapStateToProps = createStructuredSelector({
    selectedClusterId: getSelectedClusterId,
    clusters: selectors.getClusters
});

const mapDispatchToProps = dispatch => ({
    fetchClusters: () => dispatch(clusterActions.fetchClusters.request())
});

export default withRouter(
    connect(
        mapStateToProps,
        mapDispatchToProps
    )(NavigationPanel)
);
