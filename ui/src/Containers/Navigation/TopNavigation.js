import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import * as Icon from 'react-feather';
import { actions as authActions } from 'reducers/auth';

import Logo from 'Components/icons/logo';
import Menu from 'Components/Menu';
import ThemeToggleButton from 'Components/ThemeToggleButton';
import CLIDownloadButton from 'Components/CLIDownloadButton';
import GlobalSearchButton from 'Components/GlobalSearchButton';
import { useTheme } from 'Containers/ThemeProvider';
import SummaryCounts from './SummaryCounts';

const topNavBtnTextClass = 'sm:hidden md:flex uppercase text-sm tracking-wide';
const topNavBtnSvgClass = 'sm:mr-0 md:mr-3 h-4 w-4';
const topNavBtnClass =
    'flex flex-end px-4 no-underline pt-3 pb-2 text-base-600 hover:bg-base-200 items-center cursor-pointer';
const topNavMenuBtnClass =
    'no-underline text-base-600 hover:bg-base-200 items-center cursor-pointer';

const TopNavigation = ({ logout, shouldHaveReadPermission }) => {
    const { isDarkMode } = useTheme();
    function renderNavBarMenu() {
        const NavItem = () => <Icon.MoreHorizontal className="mx-4 h-4 w-4 pointer-events-none" />;
        const options = [{ label: 'Logout', onClick: () => logout() }];

        if (shouldHaveReadPermission('Licenses')) {
            options.unshift({ label: 'Product License', link: '/main/license' });
        }

        return (
            <Menu
                className={`${topNavMenuBtnClass} border-l border-base-400`}
                buttonContent={<NavItem />}
                options={options}
            />
        );
    }

    return (
        <nav
            className={`top-navigation flex flex-1 justify-between relative bg-header ${
                !isDarkMode ? 'bg-base-200' : 'bg-base-100'
            }`}
            data-test-id="top-nav-bar"
        >
            <div className="flex w-full">
                <div
                    className={`flex font-condensed font-600 uppercase py-2 px-4 border-r border-base-400 items-center ${
                        !isDarkMode ? 'bg-base-100' : 'bg-base-0'
                    }`}
                >
                    <Logo className="fill-current text-primary-800" />
                    <div className="pl-1 pt-1 text-sm tracking-wide">Platform</div>
                </div>
                <SummaryCounts />
            </div>
            <div className="flex" data-test-id="top-nav-btns">
                <GlobalSearchButton
                    topNavBtnTextClass={topNavBtnTextClass}
                    topNavBtnSvgClass={topNavBtnSvgClass}
                    topNavBtnClass={topNavBtnClass}
                />
                <CLIDownloadButton
                    topNavBtnTextClass={topNavBtnTextClass}
                    topNavBtnSvgClass={topNavBtnSvgClass}
                    topNavBtnClass={topNavBtnClass}
                />
                <ThemeToggleButton />
                {renderNavBarMenu()}
            </div>
        </nav>
    );
};

TopNavigation.propTypes = {
    logout: PropTypes.func.isRequired,
    shouldHaveReadPermission: PropTypes.func.isRequired
};

const mapStateToProps = createStructuredSelector({
    authStatus: selectors.getAuthStatus,
    shouldHaveReadPermission: selectors.shouldHaveReadPermission
});

const mapDispatchToProps = dispatch => ({
    logout: () => dispatch(authActions.logout())
});

export default withRouter(
    connect(
        mapStateToProps,
        mapDispatchToProps
    )(TopNavigation)
);
