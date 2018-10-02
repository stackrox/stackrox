import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { actions as globalSearchActions } from 'reducers/globalSearch';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import * as Icon from 'react-feather';

import Logo from 'Components/icons/logo';
import { actions as authActions, AUTH_STATUS } from 'reducers/auth';

const titleMap = {
    numClusters: { singular: 'Cluster', plural: 'Clusters' },
    numAlerts: { singular: 'Violation', plural: 'Violations' },
    numDeployments: { singular: 'Deployment', plural: 'Deployments' },
    numImages: { singular: 'Image', plural: 'Images' },
    numSecrets: { singular: 'Secret', plural: 'Secrets' }
};

class TopNavigation extends Component {
    static propTypes = {
        authStatus: PropTypes.oneOf(Object.keys(AUTH_STATUS).map(key => AUTH_STATUS[key]))
            .isRequired,
        logout: PropTypes.func.isRequired,
        toggleGlobalSearchView: PropTypes.func.isRequired,
        summaryCounts: PropTypes.shape({
            numAlerts: PropTypes.string,
            numClusters: PropTypes.string,
            numDeployments: PropTypes.string,
            numImages: PropTypes.string,
            numSecrets: PropTypes.string
        })
    };

    static defaultProps = {
        summaryCounts: null
    };

    renderLogoutButton = () => {
        if (this.props.authStatus !== AUTH_STATUS.LOGGED_IN) return null;
        return (
            <button
                onClick={this.props.logout}
                className="flex flex-end border-l border-r border-base-400 px-4 no-underline py-3 text-base-600 hover:bg-base-200 items-center cursor-pointer"
            >
                <Icon.LogOut className="h-4 w-4 mr-3" />
                <span className="uppercase text-sm tracking-wide">Logout</span>
            </button>
        );
    };

    renderSearchButton = () => (
        <button
            onClick={this.props.toggleGlobalSearchView}
            className="ignore-react-onclickoutside flex flex-end border-l border-r border-base-400 px-4 no-underline pt-3 pb-2 text-base-600 hover:bg-base-200 items-center cursor-pointer hover:bg-base-300"
        >
            <Icon.Search className="h-4 w-4 mr-3" />
            <span className="uppercase text-sm tracking-wide">Search</span>
        </button>
    );

    renderSummaryCounts = () => {
        const { summaryCounts } = this.props;
        if (!summaryCounts) return '';
        return (
            <ul className="flex uppercase text-sm p-0 w-full">
                {Object.keys(summaryCounts).map(key => (
                    <li
                        key={key}
                        className="flex flex-col border-r border-base-400 border-dashed px-3 w-24 no-underline py-3 text-base-500 items-center justify-center font-condensed"
                    >
                        <div className="text-3xl tracking-widest">{summaryCounts[key]}</div>
                        <div className="text-sm pt-1 tracking-wide">
                            {summaryCounts[key] === '1'
                                ? titleMap[key].singular
                                : titleMap[key].plural}
                        </div>
                    </li>
                ))}
            </ul>
        );
    };

    render() {
        return (
            <nav className="top-navigation flex flex-1 justify-between bg-base-200 relative bg-header">
                <div className="flex w-full">
                    <div className="flex py-2 px-4 border-r bg-base-100 border-base-400 items-center">
                        <Logo className="fill-current text-primary-800" />
                    </div>
                    {this.renderSummaryCounts()}
                </div>
                <div className="flex">
                    {this.renderSearchButton()}
                    {this.renderLogoutButton()}
                </div>
            </nav>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    summaryCounts: selectors.getSummaryCounts,
    authStatus: selectors.getAuthStatus
});

const mapDispatchToProps = dispatch => ({
    toggleGlobalSearchView: () => dispatch(globalSearchActions.toggleGlobalSearchView()),
    logout: () => dispatch(authActions.logout())
});

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(TopNavigation));
