import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { actions as globalSearchActions } from 'reducers/globalSearch';
import { actions as cliDownloadActions } from 'reducers/cli';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import * as Icon from 'react-feather';
import Tooltip from 'rc-tooltip';
import 'rc-tooltip/assets/bootstrap.css';

import Logo from 'Components/icons/logo';
import Menu from 'Components/Menu';
import { actions as authActions, AUTH_STATUS } from 'reducers/auth';

const titleMap = {
    numClusters: { singular: 'Cluster', plural: 'Clusters' },
    numNodes: { singular: 'Node', plural: 'Nodes' },
    numAlerts: { singular: 'Violation', plural: 'Violations' },
    numDeployments: { singular: 'Deployment', plural: 'Deployments' },
    numImages: { singular: 'Image', plural: 'Images' },
    numSecrets: { singular: 'Secret', plural: 'Secrets' }
};

const topNavBtnTextClass = 'sm:hidden md:flex uppercase text-sm tracking-wide';
const topNavBtnSvgClass = 'sm:mr-0 md:mr-3 h-4 w-4';
const topNavBtnClass =
    'flex flex-end px-4 no-underline pt-3 pb-2 text-base-600 hover:bg-base-200 items-center cursor-pointer';
const topNavMenuBtnClass =
    'no-underline text-base-600 hover:bg-base-200 items-center cursor-pointer';

class TopNavigation extends Component {
    static propTypes = {
        authStatus: PropTypes.oneOf(Object.keys(AUTH_STATUS).map(key => AUTH_STATUS[key]))
            .isRequired,
        logout: PropTypes.func.isRequired,
        toggleGlobalSearchView: PropTypes.func.isRequired,
        toggleCLIDownloadView: PropTypes.func.isRequired,
        summaryCounts: PropTypes.shape({
            numClusters: PropTypes.string,
            numNodes: PropTypes.string,
            numAlerts: PropTypes.string,
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
            <Tooltip
                placement="bottom"
                overlay={<div>Logout</div>}
                mouseLeaveDelay={0}
                overlayClassName="sm:visible md:invisible"
            >
                <button
                    type="button"
                    onClick={this.props.logout}
                    className={`${topNavBtnClass} border-l border-r border-base-400`}
                >
                    <Icon.LogOut className={topNavBtnSvgClass} />
                    <span className={topNavBtnTextClass}>Logout</span>
                </button>
            </Tooltip>
        );
    };

    renderSearchButton = () => (
        <Tooltip
            placement="bottom"
            overlay={<div>Search</div>}
            mouseLeaveDelay={0}
            overlayClassName="sm:visible md:invisible"
        >
            <button
                type="button"
                onClick={this.props.toggleGlobalSearchView}
                className={`${topNavBtnClass} border-l border-r border-base-400 ignore-react-onclickoutside`}
            >
                <Icon.Search className={topNavBtnSvgClass} />
                <span className={topNavBtnTextClass}>Search</span>
            </button>
        </Tooltip>
    );

    renderCLIDownloadButton = () => (
        <Tooltip
            placement="bottom"
            overlay={<div>CLI</div>}
            mouseLeaveDelay={0}
            overlayClassName="sm:visible md:invisible"
        >
            <button
                type="button"
                onClick={this.props.toggleCLIDownloadView}
                className={`${topNavBtnClass} ignore-cli-clickoutside`}
            >
                <Icon.Download className={topNavBtnSvgClass} />
                <span className={topNavBtnTextClass}>CLI</span>
            </button>
        </Tooltip>
    );

    renderSummaryCounts = () => {
        const { summaryCounts } = this.props;
        if (!summaryCounts) return '';
        return (
            <ul className="flex uppercase text-sm p-0 w-full">
                {Object.entries(titleMap).map(([key, titles]) => (
                    <li
                        key={key}
                        className="flex flex-col border-r border-base-400 border-dashed px-3 lg:w-24 md:w-20 no-underline py-3 text-base-500 items-center justify-center font-condensed"
                    >
                        <div className="text-3xl tracking-widest">{summaryCounts[key]}</div>
                        <div className="text-sm pt-1 tracking-wide">
                            {summaryCounts[key] === '1' ? titles.singular : titles.plural}
                        </div>
                    </li>
                ))}
            </ul>
        );
    };

    renderNavBarMenu = () => {
        const NavItem = () => (
            <div className="px-4">
                <Icon.MoreHorizontal className="h-4 w-4" />
            </div>
        );
        const options = [
            { label: 'Product License', link: '/main/license' },
            { label: 'Logout', onClick: () => this.props.logout() }
        ];
        return (
            <Menu
                className={`${topNavMenuBtnClass} border-l border-base-400`}
                triggerComponent={<NavItem />}
                options={options}
            />
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
                    {this.renderCLIDownloadButton()}
                    {process.env.NODE_ENV === 'development'
                        ? this.renderNavBarMenu()
                        : this.renderLogoutButton()}
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
    toggleCLIDownloadView: () => dispatch(cliDownloadActions.toggleCLIDownloadView()),
    toggleGlobalSearchView: () => dispatch(globalSearchActions.toggleGlobalSearchView()),
    logout: () => dispatch(authActions.logout())
});

export default withRouter(
    connect(
        mapStateToProps,
        mapDispatchToProps
    )(TopNavigation)
);
