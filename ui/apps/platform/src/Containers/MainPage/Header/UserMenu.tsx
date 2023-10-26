import React, { useState, CSSProperties } from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import initials from 'initials';
import { Dropdown, DropdownItem, DropdownSeparator, DropdownToggle } from '@patternfly/react-core';

import useAnalytics, { INVITE_USERS_MODAL_OPENED } from 'hooks/useAnalytics';
import usePermissions from 'hooks/usePermissions';
import { selectors } from 'reducers';
import { actions as authActions } from 'reducers/auth';
import { actions as inviteActions } from 'reducers/invite';
import { userBasePath } from 'routePaths';
import User from 'utils/User';

const userMenuStyleConstant = {
    '--pf-u-min-width--MinWidth': '20ch',
    pointerEvents: 'none',
} as CSSProperties;

function UserMenu({ logout, setInviteModalVisibility, userData }) {
    const [isOpen, setIsOpen] = useState(false);
    const { analyticsTrack } = useAnalytics();
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForInviting = hasReadWriteAccess('Access');

    function onSelect() {
        setIsOpen(false);
    }

    function onClickInviteUsers() {
        // track request to invite
        analyticsTrack(INVITE_USERS_MODAL_OPENED);

        setInviteModalVisibility(true);
    }

    const user = new User(userData);
    const { email, name, roles } = user;

    const displayName = email ? (
        <span>
            <span data-testid="menu-user-name">{name}</span> (
            <span data-testid="menu-user-email">{email}</span>)
        </span>
    ) : (
        <span data-testid="menu-user-name">{name}</span>
    );
    const displayRoles = Array.isArray(roles) ? roles.map((role) => role.name).join(',') : '';

    const startOfUserMenu = [
        <DropdownItem
            key="user"
            description={<span data-testid="menu-user-roles">{displayRoles}</span>}
            className="pf-u-min-width"
            style={userMenuStyleConstant}
        >
            {displayName}
        </DropdownItem>,
        <DropdownSeparator key="separator-1" />,
        <DropdownItem key="profile" href={userBasePath}>
            My profile
        </DropdownItem>,
    ];

    const endOfUserMenu = [
        <DropdownSeparator key="separator-2" />,
        <DropdownItem key="logout" component="button" onClick={logout}>
            Log out
        </DropdownItem>,
    ];

    const inviteMenuItem = (
        // eslint-disable-next-line @typescript-eslint/no-unsafe-return
        <DropdownItem key="open-invite" onClick={onClickInviteUsers}>
            Invite users
        </DropdownItem>
    );

    const dropdownItems = hasWriteAccessForInviting
        ? [...startOfUserMenu, inviteMenuItem, ...endOfUserMenu]
        : [...startOfUserMenu, ...endOfUserMenu];

    const toggle = (
        <DropdownToggle aria-label="User menu" onToggle={setIsOpen} toggleIndicator={null}>
            <span className="h-10 w-10 flex items-center justify-center leading-none text-xl border border-base-400 rounded-full">
                {name ? initials(name) : '--'}
            </span>
        </DropdownToggle>
    );

    return (
        <Dropdown
            dropdownItems={dropdownItems}
            isOpen={isOpen}
            isPlain
            onSelect={onSelect}
            position="right"
            toggle={toggle}
        />
    );
}

UserMenu.propTypes = {
    logout: PropTypes.func.isRequired,
    setInviteModalVisibility: PropTypes.func.isRequired,
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
};

const mapStateToProps = createStructuredSelector({
    userData: selectors.getCurrentUser,
});

const mapDispatchToProps = (dispatch) => ({
    // eslint-disable-next-line @typescript-eslint/no-unsafe-return
    setInviteModalVisibility: (show) => dispatch(inviteActions.setInviteModalVisibility(show)),
    // eslint-disable-next-line @typescript-eslint/no-unsafe-return
    logout: () => dispatch(authActions.logout()),
});

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(UserMenu));
