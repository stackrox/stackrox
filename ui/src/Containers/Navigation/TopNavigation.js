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
                className="flex flex-end border-l border-r border-base-300 px-4 no-underline py-3 text-base-600 hover:bg-base-200 items-center cursor-pointer"
            >
                <Icon.LogOut className="h-4 w-4 mr-3" />
                <span className="uppercase text-sm tracking-wide">Logout</span>
            </button>
        );
    };

    renderSearchButton = () => (
        <button
            onClick={this.props.toggleGlobalSearchView}
            className="ignore-react-onclickoutside flex flex-end border-l border-r border-base-300 px-4 no-underline py-3 text-base-600 hover:bg-base-200 items-center cursor-pointer"
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
                        className="flex flex-1 flex-col border-r border-base-300 px-4 no-underline py-3 text-base-500 items-center"
                    >
                        <div className="text-xl">{summaryCounts[key]}</div>
                        <div className="text-sm pt-1">
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
            <nav className="top-navigation flex flex-row flex-1 justify-between bg-base-100 border-base-300 border-b relative">
                <div className="flex w-full max-w-sm">
                    <div className="py-2 px-5 border-r bg-white border-base-400">
                        <Logo className="fill-current text-primary-800 h-10 w-10 " />
                    </div>
                    {this.renderSummaryCounts()}
                </div>
                <div className="flex pr-4">
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
