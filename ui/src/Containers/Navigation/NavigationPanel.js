import React, { Component } from 'react';
import { NavLink as Link, withRouter } from 'react-router-dom';
import PropTypes from 'prop-types';
import { createStructuredSelector } from 'reselect';
import { connect } from 'react-redux';

import { selectors } from 'reducers';
import {
    clustersPath,
    policiesListPath,
    integrationsPath,
    accessControlPath,
    systemConfigPath
} from 'routePaths';
import { filterLinksByFeatureFlag } from './navHelpers';

const navLinks = [
    {
        text: 'Clusters',
        to: clustersPath.replace('/:clusterId', '')
    },
    {
        text: 'System Policies',
        to: policiesListPath
    },
    {
        text: 'Integrations',
        to: integrationsPath
    },
    {
        text: 'Access Control',
        to: accessControlPath
    },
    {
        text: 'System Configuration',
        to: systemConfigPath,
        data: 'system-config'
    }
];

class NavigationPanel extends Component {
    static propTypes = {
        panelType: PropTypes.string.isRequired,
        onClose: PropTypes.func.isRequired,
        featureFlags: PropTypes.arrayOf(
            PropTypes.shape({
                envVar: PropTypes.string.isRequired,
                enabled: PropTypes.bool.isRequired
            })
        ).isRequired
    };

    constructor(props) {
        super(props);
        this.panels = {
            configure: this.renderConfigurePanel
        };
    }

    renderConfigurePanel = () => (
        <ul className="flex flex-col overflow-auto uppercase tracking-wide bg-primary-800 border-r border-l border-primary-900">
            <li className="border-b-2 border-primary-500 px-1 py-5 pl-2 pr-2 text-base-100 font-700">
                Configure StackRox Settings
            </li>
            {filterLinksByFeatureFlag(this.props.featureFlags, navLinks).map(navLink => (
                <li key={navLink.text} className="text-sm">
                    <Link
                        to={navLink.to}
                        onClick={this.props.onClose(true, 'configure')}
                        className="block no-underline text-base-100 px-1 font-700 border-b py-5 border-primary-900 pl-2 pr-2 hover:bg-base-700"
                        data-test-id={navLink.data || navLink.text}
                    >
                        {navLink.text}
                    </Link>
                </li>
            ))}
        </ul>
    );

    render() {
        return (
            <div
                className="navigation-panel w-full flex theme-light"
                data-test-id="configure-subnav"
            >
                {this.panels[this.props.panelType]()}
                <button
                    aria-label="Close Configure sub-navigation menu"
                    type="button"
                    className="flex-1 opacity-50 bg-primary-700"
                    onClick={this.props.onClose(true)}
                />
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    featureFlags: selectors.getFeatureFlags
});

export default withRouter(connect(mapStateToProps)(NavigationPanel));
