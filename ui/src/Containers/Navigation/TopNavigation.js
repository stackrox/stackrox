import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import * as Icon from 'react-feather';

import Logo from 'Components/icons/logo';
import Menu from 'Components/Menu';
import ThemeToggleButton from 'Components/ThemeToggleButton';
import CLIDownloadButton from 'Components/CLIDownloadButton';
import GlobalSearchButton from 'Components/GlobalSearchButton';
import SummaryCounts from 'Components/SummaryCounts';
import { actions as authActions } from 'reducers/auth';

export const topNavBtnTextClass = 'sm:hidden md:flex uppercase text-sm tracking-wide';
export const topNavBtnSvgClass = 'sm:mr-0 md:mr-3 h-4 w-4';
export const topNavBtnClass =
    'flex flex-end px-4 no-underline pt-3 pb-2 text-base-600 hover:bg-base-200 items-center cursor-pointer';
const topNavMenuBtnClass =
    'no-underline text-base-600 hover:bg-base-200 items-center cursor-pointer';

const TopNavigation = ({ logout, shouldHaveReadPermission }) => {
    function renderNavBarMenu() {
        const NavItem = () => <Icon.MoreHorizontal className="mx-4 h-4 w-4 pointer-events-none" />;
        const options = [{ label: 'Logout', onClick: () => logout() }];
        // dev only until feature is complete
        if (process.env.NODE_ENV !== 'production') {
            options.unshift({ label: 'System Config', link: '/main/systemconfig' });
        }
        if (shouldHaveReadPermission('Licenses')) {
            options.unshift({ label: 'Product License', link: '/main/license' });
        }
        return (
            <Menu
                className={`${topNavMenuBtnClass} border-l border-base-400`}
                triggerComponent={<NavItem />}
                options={options}
            />
        );
    }

    return (
        <nav className="top-navigation flex flex-1 justify-between bg-base-200 relative bg-header">
            <div className="flex w-full">
                <div className="flex py-2 px-4 border-r bg-base-100 border-base-400 items-center">
                    <Logo className="fill-current text-primary-800" />
                </div>
                <SummaryCounts />
            </div>
            <div className="flex">
                <GlobalSearchButton />
                <CLIDownloadButton />
                {process.env.NODE_ENV !== 'production' && <ThemeToggleButton />}
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
