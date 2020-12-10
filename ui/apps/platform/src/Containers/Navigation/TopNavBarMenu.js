import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { MoreHorizontal } from 'react-feather';
import { Avatar } from '@stackrox/ui-components';

import { selectors } from 'reducers';
import { actions as authActions } from 'reducers/auth';
import { getHasReadPermission } from 'reducers/roles';
import Menu from 'Components/Menu';
import User from 'utils/User';

const topNavMenuBtnClass =
    'no-underline text-base-600 hover:bg-base-200 items-center cursor-pointer';

function TopNavBarMenu({ logout, userRolePermissions, userData }) {
    /**
     * TODO: rework the logic for the top-right menu
     * currently starts with the last item and we conditional unshift middle item,
     * then always unshift top item
     * The use of unshift is a bit odd, especially now that the UserMenu is no longer feature-flagged.
     * Even without that, this building up of array backwards is taking the whole UI-as-code idea too far.
     *
     * Menu component should probably be adapted to just take children
     */
    const options = [{ label: 'Logout', onClick: () => logout() }];

    if (getHasReadPermission('Licenses', userRolePermissions)) {
        options.unshift({ label: 'Manage Product License', link: '/main/license' });
    }

    let buttonIcon = <MoreHorizontal className="mx-4 h-4 w-4 pointer-events-none" />;
    let buttonText = null;
    const buttonTextClassName = 'border rounded-full mx-3 p-3 text-xl border-base-400';

    const user = new User(userData);
    const { displayName } = user;
    let displayEmail = user.email;
    if (displayEmail === displayName) {
        displayEmail = null;
    }
    const menuOptionComponent = (
        <div className="flex flex-col pl-2">
            <div className="font-700 normal-case" data-testid="menu-user-name">
                {displayName}
            </div>
            {displayEmail && (
                <div className="lowercase text-base-500 italic pt-px" data-testid="menu-user-email">
                    {displayEmail}
                </div>
            )}
            <div className="pt-1" data-testid="menu-user-roles">
                <span className="font-700 pr-2">Roles ({user.roles.length}):</span>
                <span>{user.roles.map((role) => role.name).join(', ')}</span>
            </div>
        </div>
    );
    options.unshift({ component: menuOptionComponent, link: '/main/user' });
    buttonIcon = (
        <Avatar
            name={user.name || user.username}
            extraClassName="mx-3 h-10 w-10 flex items-center justify-center leading-none"
        />
    );
    buttonText = '';

    return (
        <div className="flex items-center border-l border-base-400 hover:bg-base-200">
            <Menu
                className={`${topNavMenuBtnClass} h-full`}
                menuClassName="min-w-48"
                buttonIcon={buttonIcon}
                buttonText={buttonText}
                buttonTextClassName={buttonTextClassName}
                button
                hideCaret
                options={options}
            />
        </div>
    );
}

TopNavBarMenu.propTypes = {
    logout: PropTypes.func.isRequired,
    userRolePermissions: PropTypes.shape({ globalAccess: PropTypes.string.isRequired }),
    userData: PropTypes.shape({
        userInfo: PropTypes.shape({
            username: PropTypes.string,
            friendlyName: PropTypes.string,
            roles: PropTypes.arrayOf(
                PropTypes.shape({
                    name: PropTypes.string,
                })
            ),
        }),
        userAttributes: PropTypes.arrayOf(PropTypes.shape({})),
    }).isRequired,
    // eslint-disable-next-line react/no-unused-prop-types
    featureFlags: PropTypes.arrayOf(
        PropTypes.shape({
            envVar: PropTypes.string.isRequired,
            enabled: PropTypes.bool.isRequired,
        })
    ).isRequired,
};

TopNavBarMenu.defaultProps = {
    userRolePermissions: null,
};

const mapStateToProps = createStructuredSelector({
    userRolePermissions: selectors.getUserRolePermissions,
    userData: selectors.getCurrentUser,
    featureFlags: selectors.getFeatureFlags,
});

const mapDispatchToProps = (dispatch) => ({
    logout: () => dispatch(authActions.logout()),
});

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(TopNavBarMenu));
